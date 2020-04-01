package index

import (
	context "context"
	fmt "fmt"
	"io"
	"sort"

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

	fmt.Println("Received a stat request")
	fi, err := h.fs.Stat(in.GetName())
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    uint32(fi.Mode()),
		ModTime: fi.ModTime().Unix(),
	}, nil
}

func (h *Handler) Open(stream FS_OpenServer) error {
	var fd afero.File

	for {
		r, err := stream.Recv()

		if err != nil {
			fmt.Println("Closed the request ", r, err)
			break
		}

		switch r.Request.(type) {
		case *FileRequest_Open:
			if fd != nil {
				fd.Close()
			}

			fd, err = h.fs.Open(r.GetOpen().GetName())
			if err != nil {
				return err
			}

			//  stream.Send(nil)

			defer fd.Close()
		case *FileRequest_Stat:
			fi, err := fd.Stat()
			if err != nil {
				return err
			}

			if err := stream.Send(&FileResponse{Response: &FileResponse_FileInfo{FileInfo: &FileInfo{
				Name:    fi.Name(),
				Size:    fi.Size(),
				Mode:    uint32(fi.Mode()),
				ModTime: fi.ModTime().Unix(),
			}}}); err != nil {
				return err
			}
		case *FileRequest_Truncate:
			err := fd.Truncate(r.GetTruncate().GetSize())
			if err != nil {
				return err
			}

			stream.Send(nil)
		case *FileRequest_Read:
			b := make([]byte, CHUNKSIZE)
			n, err := fd.Read(b)
			if err != nil && err != io.EOF {
				return err
			}

			stream.Send(&FileResponse{Response: &FileResponse_Read{Read: &ReadResponse{
				Content: b[:n],
			}}})
		case *FileRequest_ReadAt:
			b := make([]byte, CHUNKSIZE)
			n, err := fd.ReadAt(b, r.GetReadAt().GetOffset())
			if err != nil && err != io.EOF {
				return err
			}

			stream.Send(&FileResponse{Response: &FileResponse_Read{Read: &ReadResponse{
				Content: b[:n],
			}}})
		case *FileRequest_Readdir:
			fis, err := fd.Readdir(int(r.GetReaddir().GetCount()))
			if err != nil {
				return err
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
				return err
			}

			sort.Strings(names)

			stream.Send(&FileResponse{Response: &FileResponse_Readdirnames{Readdirnames: &ReaddirnamesResponse{
				Names: names,
			}}})
		case *FileRequest_Seek:
			offset, err := fd.Seek(r.GetSeek().GetOffset(), int(r.GetSeek().GetWhence()))
			if err != nil {
				return err
			}

			stream.Send(&FileResponse{Response: &FileResponse_Seek{Seek: &SeekResponse{
				Offset: offset,
			}}})
			stream.Send(nil)

		}
	}

	return nil
}
