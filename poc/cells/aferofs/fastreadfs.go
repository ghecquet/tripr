package aferofs

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/afero"
)

var _ afero.Lstater = (*FastReadFs)(nil)

const blockSize = 8 << 10

// unknownFileMode is a sentinel (and bogus) os.FileMode
// value used to represent a syscall.DT_UNKNOWN Dirent.Type.
const unknownFileMode os.FileMode = os.ModeNamedPipe | os.ModeSocket | os.ModeDevice

type resolver interface {
	RealPath(string) (string, error)
}

type FastReadFs struct {
	source   afero.Fs
	basepath string
}

func NewFastReadFs(source afero.Fs) afero.Fs {

	bp := "/"
	if bpfs, ok := source.(*afero.BasePathFs); ok {
		bp = afero.FullBaseFsPath(bpfs, "/")
	}

	return &FastReadFs{source: source, basepath: bp}
}

func (r *FastReadFs) ReadDir(dirName string) ([]os.FileInfo, error) {
	dirName = r.basepath + dirName

	fd, err := syscall.Open(dirName, 0, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: dirName, Err: err}
	}
	defer syscall.Close(fd)

	var fis []os.FileInfo

	// The buffer must be at least a block long.
	buf := make([]byte, blockSize) // stack-allocated; doesn't escape
	bufp := 0                      // starting read position in buf
	nbuf := 0                      // end valid data in buf

	for {
		if bufp >= nbuf {
			bufp = 0
			nbuf, err = syscall.ReadDirent(fd, buf)
			if err != nil {
				return nil, os.NewSyscallError("readdirent", err)
			}
			if nbuf <= 0 {
				return fis, nil
			}
		}
		consumed, name, typ := parseDirEnt(buf[bufp:nbuf])
		bufp += consumed
		if name == "" || name == "." || name == ".." {
			continue
		}

		// Fallback for filesystems (like old XFS) that don't
		// support Dirent.Type and have DT_UNKNOWN (0) there
		// instead.
		if typ == unknownFileMode {
			fi, err := os.Lstat(dirName + "/" + name)
			if err != nil {
				// It got deleted in the meantime.
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			typ = fi.Mode() & os.ModeType
		}

		fis = append(fis, &fastReadFileInfo{name, typ})
		// if skipFiles && typ.IsRegular() {
		// 	continue
		// }
		// if err := fn(dirName, name, typ); err != nil {
		// 	if err == ErrSkipFiles {
		// 		skipFiles = true
		// 		continue
		// 	}
		// 	return err
		// }
	}

	return fis, nil
}

func (r *FastReadFs) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

func (r *FastReadFs) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

func (r *FastReadFs) Name() string {
	return "ReadOnlyFilter"
}

func (r *FastReadFs) Stat(name string) (os.FileInfo, error) {
	return r.source.Stat(name)
}

func (r *FastReadFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	if lsf, ok := r.source.(afero.Lstater); ok {
		return lsf.LstatIfPossible(name)
	}
	fi, err := r.Stat(name)
	return fi, false, err
}

func (r *FastReadFs) Rename(o, n string) error {
	return syscall.EPERM
}

func (r *FastReadFs) RemoveAll(p string) error {
	return syscall.EPERM
}

func (r *FastReadFs) Remove(n string) error {
	return syscall.EPERM
}

func (r *FastReadFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	sourcef, err := r.source.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	return &FastReadFile{
		File: sourcef,
	}, nil
}

func (r *FastReadFs) Open(n string) (afero.File, error) {
	//fmt.Println("Opening ", n)
	sourcef, err := r.source.Open(n)
	if err != nil {
		return nil, err
	}

	return &FastReadFile{
		File: sourcef,
	}, nil
}

func (r *FastReadFs) Mkdir(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *FastReadFs) MkdirAll(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *FastReadFs) Create(n string) (afero.File, error) {
	return nil, syscall.EPERM
}

func parseDirEnt(buf []byte) (consumed int, name string, typ os.FileMode) {
	// golang.org/issue/37269
	dirent := &syscall.Dirent{}
	copy((*[unsafe.Sizeof(syscall.Dirent{})]byte)(unsafe.Pointer(dirent))[:], buf)
	if v := unsafe.Offsetof(dirent.Reclen) + unsafe.Sizeof(dirent.Reclen); uintptr(len(buf)) < v {
		panic(fmt.Sprintf("buf size of %d smaller than dirent header size %d", len(buf), v))
	}
	if len(buf) < int(dirent.Reclen) {
		panic(fmt.Sprintf("buf size %d < record length %d", len(buf), dirent.Reclen))
	}
	consumed = int(dirent.Reclen)
	if direntInode(dirent) == 0 { // File absent in directory.
		return
	}

	switch dirent.Type {
	case syscall.DT_REG:
		typ = 0
	case syscall.DT_DIR:
		typ = os.ModeDir
	case syscall.DT_LNK:
		typ = os.ModeSymlink
	case syscall.DT_BLK:
		typ = os.ModeDevice
	case syscall.DT_FIFO:
		typ = os.ModeNamedPipe
	case syscall.DT_SOCK:
		typ = os.ModeSocket
	case syscall.DT_UNKNOWN:
		typ = unknownFileMode
	default:
		// Skip weird things.
		// It's probably a DT_WHT (http://lwn.net/Articles/325369/)
		// or something. Revisit if/when this package is moved outside
		// of goimports. goimports only cares about regular files,
		// symlinks, and directories.
		return
	}

	nameBuf := (*[unsafe.Sizeof(dirent.Name)]byte)(unsafe.Pointer(&dirent.Name[0]))
	nameLen := direntNamlen(dirent)

	// Special cases for common things:
	if nameLen == 1 && nameBuf[0] == '.' {
		name = "."
	} else if nameLen == 2 && nameBuf[0] == '.' && nameBuf[1] == '.' {
		name = ".."
	} else {
		name = string(nameBuf[:nameLen])
	}
	return
}

type FastReadFile struct {
	afero.File
}

type fdable interface {
	Fd() uintptr
}

func (f *FastReadFile) Readdir(count int) ([]os.FileInfo, error) {
	//fmt.Println("Reading dir ")
	fd := uintptr(afero.BADFD)
	if ffd, ok := f.File.(fdable); ok {
		fd = ffd.Fd()
	}

	if fd == uintptr(afero.BADFD) {
		return f.File.Readdir(count)
	}

	var fis []os.FileInfo
	var err error

	// The buffer must be at least a block long.
	buf := make([]byte, blockSize) // stack-allocated; doesn't escape
	bufp := 0                      // starting read position in buf
	nbuf := 0                      // end valid data in buf

	for {
		if bufp >= nbuf {
			bufp = 0
			nbuf, err = syscall.ReadDirent(int(fd), buf)
			if err != nil {
				return nil, os.NewSyscallError("readdirent", err)
			}
			if nbuf <= 0 {
				return fis, nil
			}
		}
		consumed, name, typ := parseDirEnt(buf[bufp:nbuf])
		bufp += consumed
		if name == "" || name == "." || name == ".." {
			continue
		}

		// Fallback for filesystems (like old XFS) that don't
		// support Dirent.Type and have DT_UNKNOWN (0) there
		// instead.
		// if typ == unknownFileMode {
		// 	fi, err := os.Lstat(dirName + "/" + name)
		// 	if err != nil {
		// 		// It got deleted in the meantime.
		// 		if os.IsNotExist(err) {
		// 			continue
		// 		}
		// 		return nil, err
		// 	}
		// 	typ = fi.Mode() & os.ModeType
		// }

		fis = append(fis, &fastReadFileInfo{name, typ})
		// if skipFiles && typ.IsRegular() {
		// 	continue
		// }
		// if err := fn(dirName, name, typ); err != nil {
		// 	if err == ErrSkipFiles {
		// 		skipFiles = true
		// 		continue
		// 	}
		// 	return err
		// }
	}

	return fis, nil
}

type fastReadFileInfo struct {
	name string
	mode os.FileMode
}

func (f *fastReadFileInfo) IsDir() bool {
	return (f.mode & os.ModeType).IsDir()
}

func (f *fastReadFileInfo) Name() string {
	return f.name
}

func (f *fastReadFileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *fastReadFileInfo) ModTime() time.Time {
	return time.Unix(0, 0)
}

func (f *fastReadFileInfo) Size() int64 {
	return 0
}

func (f *fastReadFileInfo) Sys() interface{} {
	return nil
}
