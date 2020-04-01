// Copyright © 2015 Jerry Jacobs <jerry.jacobs@xor-gate.org>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	context "context"
	fmt "fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

type File struct {
	name   string
	stream FS_OpenClient
}

type fileInfo struct {
	*FileInfo
}

func NewIndexFile(ctx context.Context, name string, cli FSClient) afero.File {
	stream, _ := cli.Open(ctx)

	// Sending initial request to open the file descriptor
	stream.Send(&FileRequest{
		Request: &FileRequest_Open{Open: &OpenRequest{Name: name}},
	})

	// stream.Recv()

	return &File{
		name:   name,
		stream: stream,
	}
}

func (f *File) Close() error {
	if f.stream == nil {
		return nil
	}
	return f.stream.CloseSend()
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Stat() (os.FileInfo, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Stat{Stat: &StatRequest{}},
	})
	if err != nil {
		return nil, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return nil, err
	}

	return &fileInfo{resp.GetFileInfo()}, nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Truncate{Truncate: &TruncateRequest{Size: size}},
	})
	if err != nil {
		return err
	}

	if _, err := f.stream.Recv(); err != nil {
		return err
	}

	return nil
}

func (f *File) Read(b []byte) (int, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Read{Read: &ReadRequest{}},
	})
	if err != nil {
		return 0, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, err
	}

	n := copy(b, resp.GetRead().GetContent())

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (f *File) ReadAt(b []byte, off int64) (int, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_ReadAt{ReadAt: &ReadAtRequest{Offset: off}},
	})
	if err != nil {
		return 0, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, err
	}

	n := copy(resp.GetRead().GetContent(), b)

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Readdir{Readdir: &ReaddirRequest{}},
	})
	if err != nil {
		return nil, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// fmt.Println("Received a resp ", resp)

	var fis []os.FileInfo
	for _, fi := range resp.GetReaddir().GetFileInfo() {
		fis = append(fis, &fileInfo{fi})
	}

	return fis, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Readdirnames{Readdirnames: &ReaddirnamesRequest{}},
	})
	if err != nil {
		return nil, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return nil, err
	}

	return resp.GetReaddirnames().GetNames(), nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	err := f.stream.Send(&FileRequest{
		Request: &FileRequest_Seek{Seek: &SeekRequest{Offset: offset, Whence: SeekRequest_Whence(whence)}},
	})
	if err != nil {
		return 0, err
	}

	resp, err := f.stream.Recv()
	if err != nil {
		return 0, err
	}

	return resp.GetSeek().GetOffset(), nil
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, syscall.EPERM
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, syscall.EPERM
}

func (f *File) WriteString(s string) (ret int, err error) {
	return 0, syscall.EPERM
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
