package aferofs

import (
	context "context"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	_ "github.com/ghecquet/tripr/poc/cells/client/resolver"
	"github.com/ghecquet/tripr/poc/cells/index"
	"github.com/spf13/afero"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var _ afero.Lstater = (*IndexFs)(nil)

type IndexFs struct {
	ctx context.Context
	cli index.FSClient
}

func NewIndexFs(path string) afero.Fs {
	selector := strings.Split(path, "@")

	// TODO - Need some type of selector
	conn, err := grpc.Dial("cells:///"+selector[0]+"@index.FS", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// defer conn.Close()

	c := index.NewFSClient(conn)
	ctx := context.TODO()

	return afero.NewBasePathFs(&IndexFs{
		ctx: ctx,
		cli: c,
	}, selector[1])
}

func (f *IndexFs) ReadDir(name string) ([]os.FileInfo, error) {
	return afero.ReadDir(f, name)
}

func (f *IndexFs) Chtimes(name string, added, modified time.Time) error {
	_, err := f.cli.Chtimes(f.ctx, &index.ChtimesRequest{
		Name:     name,
		Added:    added.Unix(),
		Modified: modified.Unix(),
	})

	return err
}

func (f *IndexFs) Chmod(name string, mode os.FileMode) error {
	_, err := f.cli.Chmod(f.ctx, &index.ChmodRequest{
		Name: name,
		Mode: uint32(mode),
	})

	return err
}

func (f *IndexFs) Name() string {
	return "IndexFs"
}

func (f *IndexFs) Stat(name string) (os.FileInfo, error) {
	fi, err := f.cli.Stat(f.ctx, &index.FileRequest{
		Request: &index.FileRequest_Name{name},
	})
	if err != nil {
		return nil, err
	}

	return &fileInfo{fi}, nil
}

func (f *IndexFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := f.cli.Stat(f.ctx, &index.FileRequest{
		Request: &index.FileRequest_Name{name},
	})
	if err != nil {
		return nil, false, err
	}

	return &fileInfo{fi}, false, nil
}

func (f *IndexFs) Rename(oldName, newName string) error {
	_, err := f.cli.Rename(f.ctx, &index.RenameRequest{
		OldName: oldName,
		NewName: newName,
	})

	return err
}

func (f *IndexFs) RemoveAll(path string) error {
	_, err := f.cli.RemoveAll(f.ctx, &index.RemoveAllRequest{
		Path: path,
	})

	return err
}

func (f *IndexFs) Remove(name string) error {
	_, err := f.cli.Remove(f.ctx, &index.RemoveRequest{
		Name: name,
	})

	return err
}

func (f *IndexFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return NewIndexFile(f.ctx, name, flag, perm, f.cli)
}

func (f *IndexFs) Open(name string) (afero.File, error) {
	return f.OpenFile(name, os.O_RDONLY, 0)
}

func (f *IndexFs) Mkdir(name string, perm os.FileMode) error {
	_, err := f.cli.Mkdir(f.ctx, &index.MkdirRequest{
		Name: name,
		Perm: uint32(perm),
	})

	return err
}

func (f *IndexFs) MkdirAll(path string, perm os.FileMode) error {
	_, err := f.cli.MkdirAll(f.ctx, &index.MkdirAllRequest{
		Path: path,
		Perm: uint32(perm),
	})

	return err
}

func (f *IndexFs) Create(name string) (afero.File, error) {
	return f.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

type File struct {
	name   string
	stream index.FS_OpenClient
}

func NewIndexFile(ctx context.Context, name string, flag int, perm os.FileMode, cli index.FSClient) (afero.File, error) {
	stream, err := cli.Open(ctx)
	if err != nil {
		return nil, fromRPCError(err)
	}

	// Sending initial request to open the file descriptor
	if err := stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Open{Open: &index.OpenRequest{
			Name:     name,
			Flag:     int64(flag),
			FileMode: uint32(perm),
		}},
	}); err != nil {
		return nil, fromRPCError(err)
	}

	if _, err := stream.Recv(); err != nil {
		return nil, fromRPCError(err)
	}

	return &File{
		name:   name,
		stream: stream,
	}, nil
}

func (f *File) Close() error {
	if f.stream == nil {
		return nil
	}
	err := f.stream.CloseSend()
	if err != nil {
		return fromRPCError(err)
	}
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Stat() (os.FileInfo, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Stat{Stat: &index.StatRequest{}},
	})
	if err != nil {
		return nil, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return nil, fromRPCError(err)
	}

	return &fileInfo{resp.GetFileInfo()}, nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Truncate{Truncate: &index.TruncateRequest{Size: size}},
	})
	if err != nil {
		return fromRPCError(err)
	}

	if _, err := f.stream.Recv(); err != nil {
		return fromRPCError(err)
	}

	return nil
}

func (f *File) Read(b []byte) (int, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Read{Read: &index.ReadRequest{}},
	})
	if err != nil {
		return 0, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, fromRPCError(err)
	}

	n := copy(b, resp.GetRead().GetContent())

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (f *File) ReadAt(b []byte, off int64) (int, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_ReadAt{ReadAt: &index.ReadAtRequest{Offset: off}},
	})
	if err != nil {
		return 0, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, fromRPCError(err)
	}

	n := copy(resp.GetRead().GetContent(), b)

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Readdir{Readdir: &index.ReaddirRequest{}},
	})
	if err != nil {
		return nil, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return nil, fromRPCError(err)
	}

	var fis []os.FileInfo
	for _, fi := range resp.GetReaddir().GetFileInfo() {
		fis = append(fis, &fileInfo{fi})
	}

	return fis, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Readdirnames{Readdirnames: &index.ReaddirnamesRequest{}},
	})
	if err != nil {
		return nil, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return nil, fromRPCError(err)
	}

	return resp.GetReaddirnames().GetNames(), nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Seek{Seek: &index.SeekRequest{Offset: offset, Whence: index.SeekRequest_Whence(whence)}},
	})
	if err != nil {
		return 0, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, fromRPCError(err)
	}

	return resp.GetSeek().GetOffset(), nil
}

func (f *File) Write(b []byte) (int, error) {
	err := f.stream.Send(&index.FileRequest{
		Request: &index.FileRequest_Write{Write: &index.WriteRequest{Content: b}},
	})
	if err != nil {
		return 0, fromRPCError(err)
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, fromRPCError(err)
	}

	return int(resp.GetWrite().GetBytesWritten()), nil
}

func (f *File) WriteAt(b []byte, off int64) (int, error) {
	n := 0

	for len(b) > 0 {
		err := f.stream.Send(&index.FileRequest{
			Request: &index.FileRequest_WriteAt{WriteAt: &index.WriteAtRequest{Content: b, Offset: off}},
		})
		if err != nil {
			return 0, fromRPCError(err)
		}

		resp, err := f.stream.Recv()
		if err != nil {
			return 0, fromRPCError(err)
		}

		m := resp.GetWrite().GetBytesWritten()
		n += int(m)
		b = b[m:]
		off += m
	}

	return n, nil
}

func (f *File) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

func fromRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		// Error was not a status error
		return err
	}

	switch msg := st.Message(); msg {
	case syscall.ENOENT.Error():
		return syscall.ENOENT
	}

	return errors.New(st.Message())
}

type fileInfo struct {
	*index.FileInfo
}

func (f *fileInfo) IsDir() bool {
	return f.GetIsDir()
}

func (f *fileInfo) Name() string {
	return f.GetName()
}

func (f *fileInfo) Mode() os.FileMode {
	return os.FileMode(f.GetMode())
}

func (f *fileInfo) ModTime() time.Time {
	return time.Unix(f.GetModTime(), 0)
}

func (f *fileInfo) Size() int64 {
	return f.GetSize()
}

func (f *fileInfo) Sys() interface{} {
	return nil
}
