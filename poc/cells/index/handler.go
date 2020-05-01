package index

import (
	context "context"
	"io"
	"os"
	"sort"
	"time"

	"github.com/spf13/afero"
)

const CHUNKSIZE = 1024

type Handler struct {
	fs afero.Fs
}

func NewHandler(fs afero.Fs) *Handler {
	return &Handler{
		fs: fs,
	}
}

func (h *Handler) Stat(ctx context.Context, in *FileRequest) (*FileInfo, error) {
	fi, err := h.fs.Stat(in.GetName())
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    uint32(fi.Mode()),
		ModTime: fi.ModTime().Unix(),
		IsDir:   fi.IsDir(),
	}, nil
}

func (h *Handler) Chtimes(ctx context.Context, in *ChtimesRequest) (*ChtimesResponse, error) {
	err := h.fs.Chtimes(in.Name, time.Unix(in.Added, 0), time.Unix(in.Modified, 0))

	return &ChtimesResponse{}, err
}

func (h *Handler) Chmod(ctx context.Context, in *ChmodRequest) (*ChmodResponse, error) {
	err := h.fs.Chmod(in.Name, os.FileMode(in.Mode))

	return &ChmodResponse{}, err
}

func (h *Handler) Mkdir(ctx context.Context, in *MkdirRequest) (*MkdirResponse, error) {
	err := h.fs.Mkdir(in.Name, os.FileMode(in.Perm))

	return &MkdirResponse{}, err
}

func (h *Handler) MkdirAll(ctx context.Context, in *MkdirAllRequest) (*MkdirAllResponse, error) {
	err := h.fs.MkdirAll(in.Path, os.FileMode(in.Perm))

	return &MkdirAllResponse{}, err
}

func (h *Handler) Rename(ctx context.Context, in *RenameRequest) (*RenameResponse, error) {
	err := h.fs.Rename(in.OldName, in.NewName)

	return &RenameResponse{}, err
}

func (h *Handler) RemoveAll(ctx context.Context, in *RemoveAllRequest) (*RemoveAllResponse, error) {
	err := h.fs.RemoveAll(in.Path)

	return &RemoveAllResponse{}, err
}

func (h *Handler) Remove(ctx context.Context, in *RemoveRequest) (*RemoveResponse, error) {
	err := h.fs.Remove(in.Name)

	return &RemoveResponse{}, err
}

func (h *Handler) Open(stream FS_OpenServer) error {
	var fd afero.File

	for {
		r, err := stream.Recv()

		if err != nil {
			break
		}

		switch r.Request.(type) {
		case *FileRequest_Open:
			if fd != nil {
				fd.Close()
			}

			in := r.GetOpen()

			fd, err = h.fs.OpenFile(in.GetName(), int(in.GetFlag()), os.FileMode(in.GetFileMode()))
			if err != nil {
				return getError(err)
			}

			if err := stream.Send(&FileResponse{Response: &FileResponse_Open{Open: &OpenResponse{}}}); err != nil {
				return getError(err)
			}

			defer fd.Close()
		case *FileRequest_Stat:
			fi, err := fd.Stat()
			if err != nil {
				return getError(err)
			}

			if err := stream.Send(&FileResponse{Response: &FileResponse_FileInfo{FileInfo: &FileInfo{
				Name:    fi.Name(),
				Size:    fi.Size(),
				Mode:    uint32(fi.Mode()),
				ModTime: fi.ModTime().Unix(),
			}}}); err != nil {
				return getError(err)
			}
		case *FileRequest_Truncate:
			err := fd.Truncate(r.GetTruncate().GetSize())
			if err != nil {
				return getError(err)
			}

			stream.Send(nil)
		case *FileRequest_Read:
			b := make([]byte, CHUNKSIZE)
			n, err := fd.Read(b)
			if err != nil && err != io.EOF {
				return getError(err)
			}

			stream.Send(&FileResponse{Response: &FileResponse_Read{Read: &ReadResponse{
				Content: b[:n],
			}}})
		case *FileRequest_ReadAt:
			b := make([]byte, CHUNKSIZE)
			n, err := fd.ReadAt(b, r.GetReadAt().GetOffset())
			if err != nil && err != io.EOF {
				return getError(err)
			}

			stream.Send(&FileResponse{Response: &FileResponse_Read{Read: &ReadResponse{
				Content: b[:n],
			}}})
		case *FileRequest_Readdir:
			fis, err := fd.Readdir(int(r.GetReaddir().GetCount()))
			if err != nil {
				return getError(err)
			}

			var ret []*FileInfo
			for _, fi := range fis {
				ret = append(ret, &FileInfo{
					Name:    fi.Name(),
					Size:    fi.Size(),
					Mode:    uint32(fi.Mode()),
					ModTime: fi.ModTime().Unix(),
					IsDir:   fi.IsDir(),
				})
			}

			sort.Sort(byName(ret))

			stream.Send(&FileResponse{Response: &FileResponse_Readdir{Readdir: &ReaddirResponse{
				FileInfo: ret,
			}}})
		case *FileRequest_Readdirnames:
			names, err := fd.Readdirnames(int(r.GetReaddirnames().GetCount()))
			if err != nil {
				return getError(err)
			}

			sort.Strings(names)

			stream.Send(&FileResponse{Response: &FileResponse_Readdirnames{Readdirnames: &ReaddirnamesResponse{
				Names: names,
			}}})
		case *FileRequest_Seek:
			offset, err := fd.Seek(r.GetSeek().GetOffset(), int(r.GetSeek().GetWhence()))
			if err != nil {
				return getError(err)
			}

			stream.Send(&FileResponse{Response: &FileResponse_Seek{Seek: &SeekResponse{
				Offset: offset,
			}}})
		case *FileRequest_Write:
			bytesWritten, err := fd.Write(r.GetWrite().GetContent())
			if err != nil {
				return getError(err)
			}

			stream.Send(&FileResponse{Response: &FileResponse_Write{Write: &WriteResponse{
				BytesWritten: int64(bytesWritten),
			}}})

		case *FileRequest_WriteAt:
			bytesWritten, err := fd.WriteAt(r.GetWriteAt().GetContent(), r.GetWriteAt().GetOffset())
			if err != nil {
				return getError(err)
			}

			stream.Send(&FileResponse{Response: &FileResponse_Write{Write: &WriteResponse{
				BytesWritten: int64(bytesWritten),
			}}})
		}
	}

	return nil
}

func getError(err error) error {
	switch v := err.(type) {
	case *os.PathError:
		return v.Err
	}
	return err
}

// byName implements sort.Interface.
type byName []*FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name < f[j].Name }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
