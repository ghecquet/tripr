package index

import (
	context "context"
	"log"
	"os"
	"syscall"
	"time"

	_ "github.com/ghecquet/tripr/poc/cells/client/resolver"
	"github.com/spf13/afero"
	grpc "google.golang.org/grpc"
)

//var _ Lstater = (*IndexFs)(nil)

type Fs struct {
	ctx context.Context
	cli FSClient
}

func NewFs() afero.Fs {
	// TODO - Need some type of selector
	conn, err := grpc.Dial("cells:///index.FS", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// defer conn.Close()

	c := NewFSClient(conn)
	ctx := context.TODO()

	return &Fs{
		ctx: ctx,
		cli: c,
	}
}

func (f *Fs) ReadDir(name string) ([]os.FileInfo, error) {
	return afero.ReadDir(f, name)
}

func (f *Fs) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

func (f *Fs) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

func (f *Fs) Name() string {
	return "IndexFs"
}

func (f *Fs) Stat(name string) (os.FileInfo, error) {
	fi, err := f.cli.Stat(f.ctx, &FileRequest{
		Request: &FileRequest_Name{name},
	})
	if err != nil {
		return nil, err
	}

	return &fileInfo{fi}, nil
}

func (f *Fs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := f.cli.Stat(f.ctx, &FileRequest{
		Request: &FileRequest_Name{name},
	})
	if err != nil {
		return nil, false, err
	}

	return &fileInfo{fi}, false, nil
}

func (f *Fs) Rename(o, n string) error {
	return syscall.EPERM
}

func (f *Fs) RemoveAll(p string) error {
	return syscall.EPERM
}

func (f *Fs) Remove(n string) error {
	return syscall.EPERM
}

func (f *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	return NewIndexFile(f.ctx, name, f.cli), nil
}

func (f *Fs) Open(name string) (afero.File, error) {
	return NewIndexFile(f.ctx, name, f.cli), nil
}

func (f *Fs) Mkdir(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (f *Fs) MkdirAll(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (f *Fs) Create(n string) (afero.File, error) {
	return nil, syscall.EPERM
}
