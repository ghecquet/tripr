package compositefs

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

type CompositeFs struct {
	fs1 afero.Fs
	fs2 afero.Fs
}

func NewCompositeFs(fs1 afero.Fs, fs2 afero.Fs) afero.Fs {
	return &CompositeFs{fs1: fs1, fs2: fs2}
}

type cacheState int

const (
	// not present in any of the fs
	cacheMiss cacheState = iota

	// not present in first fs
	cacheMiss1

	// not present in second fs
	cacheMiss2

	// present in both, but fs 1 has a more recent version
	cacheHit1

	// present in both, but fs 2 has a more recent version
	cacheHit2

	// present in one of the fs
	cacheHit
)

func (u *CompositeFs) cacheStatus(name string) (cacheState, os.FileInfo, error) {
	fs1fi, err1 := u.fs1.Stat(name)
	fs2fi, err2 := u.fs2.Stat(name)

	if err1 == nil && err2 == nil {
		fs1modtime := fs1fi.ModTime()
		fs2modtime := fs2fi.ModTime()

		if fs1modtime.Equal(fs2modtime) {
			return cacheHit, fs1fi, nil
		} else if fs1modtime.Before(fs2modtime) {
			return cacheHit2, fs2fi, nil
		} else {
			return cacheHit1, fs1fi, nil
		}
	}

	if err1 == syscall.ENOENT || os.IsNotExist(err1) {
		if err2 == syscall.ENOENT || os.IsNotExist(err2) {
			return cacheMiss, nil, nil
		}

		return cacheMiss1, nil, nil
	}

	if err2 == syscall.ENOENT || os.IsNotExist(err2) {
		return cacheMiss2, nil, nil
	}

	return cacheMiss, nil, err1
}

// func (u *CompositeFs) copyToLayer(name string) error {
// 	return copyToLayer(u.base, u.layer, name)
// }

func (u *CompositeFs) Chtimes(name string, atime, mtime time.Time) error {
	st, _, err := u.cacheStatus(name)
	if err != nil {
		return err
	}
	switch st {
	case cacheHit:
	case cacheHit1:
	case cacheHit2:
		err1 := u.fs1.Chtimes(name, atime, mtime)
		err2 := u.fs2.Chtimes(name, atime, mtime)
		if err1 != nil || err2 != nil {
			return err1
		}
	case cacheMiss2:
		err = u.fs1.Chtimes(name, atime, mtime)
	case cacheMiss1:
		err = u.fs2.Chtimes(name, atime, mtime)
	case cacheMiss:
		err = fmt.Errorf("no entry")
	}
	return err
}

func (u *CompositeFs) Chmod(name string, mode os.FileMode) error {
	st, _, err := u.cacheStatus(name)
	if err != nil {
		return err
	}
	switch st {
	case cacheHit:
	case cacheHit1:
	case cacheHit2:
		err1 := u.fs1.Chmod(name, mode)
		err2 := u.fs2.Chmod(name, mode)
		if err1 != nil || err2 != nil {
			return err1
		}
	case cacheMiss2:
		err = u.fs1.Chmod(name, mode)
	case cacheMiss1:
		err = u.fs2.Chmod(name, mode)
	case cacheMiss:
		err = fmt.Errorf("no entry")
	}

	return err
}

func (u *CompositeFs) Stat(name string) (os.FileInfo, error) {
	st, fi, err := u.cacheStatus(name)
	if err != nil {
		return nil, err
	}
	switch st {
	case cacheMiss:
		return nil, fmt.Errorf("not found")
	default: // file info is managed in cacheStatus
		return fi, nil
	}
}

func (u *CompositeFs) Rename(oldname, newname string) error {
	st, _, err := u.cacheStatus(oldname)
	if err != nil {
		return err
	}
	switch st {
	case cacheHit:
	case cacheHit1:
	case cacheHit2:
		err1 := u.fs1.Rename(oldname, newname)
		err2 := u.fs2.Rename(oldname, newname)
		if err1 != nil || err2 != nil {
			return err1
		}
	case cacheMiss2:
		err = u.fs1.Rename(oldname, newname)
	case cacheMiss1:
		err = u.fs2.Rename(oldname, newname)
	case cacheMiss:
		err = fmt.Errorf("no entry")
	}
	return err
}

func (u *CompositeFs) Remove(name string) error {
	st, _, err := u.cacheStatus(name)
	if err != nil {
		return err
	}
	switch st {
	case cacheHit:
	case cacheHit1:
	case cacheHit2:
		err1 := u.fs1.Remove(name)
		err2 := u.fs2.Remove(name)
		if err1 != nil || err2 != nil {
			return err1
		}
	case cacheMiss2:
		err = u.fs1.Remove(name)
	case cacheMiss1:
		err = u.fs2.Remove(name)
	case cacheMiss:
		err = fmt.Errorf("no entry")
	}
	return err
}

func (u *CompositeFs) RemoveAll(name string) error {
	st, _, err := u.cacheStatus(name)
	if err != nil {
		return err
	}

	switch st {
	case cacheHit:
	case cacheHit1:
	case cacheHit2:
		err1 := u.fs1.RemoveAll(name)
		err2 := u.fs2.RemoveAll(name)
		if err1 != nil || err2 != nil {
			return err1
		}
	case cacheMiss2:
		err = u.fs1.RemoveAll(name)
	case cacheMiss1:
		err = u.fs2.RemoveAll(name)
	case cacheMiss:
		err = fmt.Errorf("no entry")
	}
	return err
}

func (u *CompositeFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	// st, _, err := u.cacheStatus(name)
	// if err != nil {
	// 	return nil, err
	// }
	fs1fi, err := u.fs1.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	fs2fi, err := u.fs2.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return &afero.UnionFile{Base: fs1fi, Layer: fs2fi}, nil
}

func (u *CompositeFs) Open(name string) (afero.File, error) {
	// st, fi, err := u.cacheStatus(name)
	// if err != nil {
	// 	return nil, err
	// }

	fmt.Println("Opening this ", name)

	f1, err := u.fs1.Open(name)
	if err != nil {
		return nil, err
	}
	f2, err := u.fs2.Open(name)
	if err != nil {
		return nil, err
	}
	return &afero.UnionFile{Base: f1, Layer: f2}, nil
}

func (u *CompositeFs) Mkdir(name string, perm os.FileMode) error {
	// yes, MkdirAll... we cannot assume it exists in both fs
	err1 := u.fs1.MkdirAll(name, perm)
	err2 := u.fs2.MkdirAll(name, perm)

	if err1 != nil || err2 != nil {
		return err1
	}

	return nil
}

func (u *CompositeFs) Name() string {
	return "CompositeFs"
}

func (u *CompositeFs) MkdirAll(name string, perm os.FileMode) error {
	err1 := u.fs1.MkdirAll(name, perm)
	err2 := u.fs2.MkdirAll(name, perm)

	if err1 != nil || err2 != nil {
		return err1
	}

	return nil
}

func (u *CompositeFs) Create(name string) (afero.File, error) {
	f1, err1 := u.fs1.Create(name)
	f2, err2 := u.fs2.Create(name)

	if err1 != nil || err2 != nil {
		return nil, err1
	}

	return &afero.UnionFile{Base: f1, Layer: f2}, nil
}
