package index

import (
	"os"
	"time"
)

type fileInfo struct {
	*FileInfo
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

// byName implements sort.Interface.
type byName []*FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name < f[j].Name }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
