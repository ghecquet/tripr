package billyfs

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/helper/chroot"
	"gopkg.in/src-d/go-billy.v4/util"
)

const (
	defaultDirectoryMode = 0755
	defaultCreateMode    = 0666
)

type AferoFs struct {
	afero.Fs
}

type file struct {
	afero.File
	m sync.Mutex
}

// New returns a new OS filesystem.
func NewAfero(fs afero.Fs) billy.Filesystem {
	return chroot.New(&AferoFs{fs}, "/")
}

func (f *AferoFs) Create(filename string) (billy.File, error) {
	return f.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, defaultCreateMode)
}

func (f *AferoFs) Open(filename string) (billy.File, error) {
	return f.OpenFile(filename, os.O_RDONLY, 0)
}

func (f *AferoFs) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if flag&os.O_CREATE != 0 {
		if err := f.createDir(filename); err != nil {
			return nil, err
		}
	}

	fd, err := f.Fs.OpenFile(filename, flag, perm)
	if err != nil {
		return nil, err
	}
	return &file{File: fd}, err
}

func (f *AferoFs) createDir(fullpath string) error {
	dir := filepath.Dir(fullpath)
	if dir != "." {
		if err := f.Fs.MkdirAll(dir, defaultDirectoryMode); err != nil {
			return err
		}
	}

	return nil
}

func (f *AferoFs) Stat(filename string) (os.FileInfo, error) {
	return f.Fs.Stat(filename)
}

func (f *AferoFs) Rename(oldpath, newpath string) error {
	if err := f.createDir(newpath); err != nil {
		return err
	}

	return f.Fs.Rename(oldpath, newpath)
}

func (f *AferoFs) Remove(filename string) error {
	return f.Fs.Remove(filename)
}

func (f *AferoFs) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (f *AferoFs) TempFile(dir, prefix string) (billy.File, error) {
	if err := f.createDir(dir + string(os.PathSeparator)); err != nil {
		return nil, err
	}

	fd, err := afero.TempFile(f.Fs, dir, prefix)
	if err != nil {
		return nil, err
	}
	return &file{File: fd}, nil
}

func (f *AferoFs) ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(f.Fs, path)
}

func (f *AferoFs) MkdirAll(filename string, perm os.FileMode) error {
	return f.Fs.MkdirAll(filename, perm)
}

func (f *AferoFs) Lstat(filename string) (os.FileInfo, error) {
	lstater, ok := (f.Fs).(afero.Lstater)
	if ok {
		fi, _, err := lstater.LstatIfPossible(filename)
		return fi, err
	}

	return f.Stat(filename)
}

func (f *AferoFs) Symlink(target, link string) error {
	_, err := f.Stat(link)
	if err == nil {
		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return err
	}

	return util.WriteFile(f, link, []byte(target), 0777|os.ModeSymlink)
}

func (f *AferoFs) Readlink(link string) (string, error) {
	// f, has := fs.s.Get(link)
	// if !has {
	// 	return "", os.ErrNotExist
	// }

	// if !isSymlink(f.mode) {
	// 	return "", &os.PathError{
	// 		Op:   "readlink",
	// 		Path: link,
	// 		Err:  fmt.Errorf("not a symlink"),
	// 	}
	// }

	return "", nil
}

func (f *file) Lock() error {
	f.m.Lock()
	defer f.m.Unlock()

	return nil
}

func (f *file) Unlock() error {
	f.m.Lock()
	defer f.m.Unlock()

	return nil
}
