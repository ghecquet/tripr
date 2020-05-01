// Package local provides a filesystem interface
package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/ghecquet/tripr/poc/cells/index"
	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/hash"
	"github.com/spf13/afero"
)

// Constants
const devUnset = 0xdeadbeefcafebabe                                       // a device id meaning it is unset
const linkSuffix = ".rclonelink"                                          // The suffix added to a translated symbolic link
const useReadDir = (runtime.GOOS == "windows" || runtime.GOOS == "plan9") // these OSes read FileInfos directly

// Register with Fs
func init() {
	fsi := &fs.RegInfo{
		Name:        "fs",
		Description: "File System",
		NewFs:       NewFs,
		Options: []fs.Option{{
			Name: "nounc",
			Help: "Disable UNC (long path names) conversion on Windows",
			Examples: []fs.OptionExample{{
				Value: "true",
				Help:  "Disables long file names",
			}},
		}, {
			Name:     "copy_links",
			Help:     "Follow symlinks and copy the pointed to item.",
			Default:  false,
			NoPrefix: true,
			ShortOpt: "L",
			Advanced: true,
		}, {
			Name:     "links",
			Help:     "Translate symlinks to/from regular files with a '" + linkSuffix + "' extension",
			Default:  false,
			NoPrefix: true,
			ShortOpt: "l",
			Advanced: true,
		}, {
			Name: "skip_links",
			Help: `Don't warn about skipped symlinks.
This flag disables warning messages on skipped symlinks or junction
points, as you explicitly acknowledge that they should be skipped.`,
			Default:  false,
			NoPrefix: true,
			Advanced: true,
		}, {
			Name: "no_unicode_normalization",
			Help: `Don't apply unicode normalization to paths and filenames (Deprecated)

This flag is deprecated now.  Rclone no longer normalizes unicode file
names, but it compares them with unicode normalization in the sync
routine instead.`,
			Default:  false,
			Advanced: true,
		}, {
			Name: "no_check_updated",
			Help: `Don't check to see if the files change during upload

Normally rclone checks the size and modification time of files as they
are being uploaded and aborts with a message which starts "can't copy
- source file is being updated" if the file changes during upload.

However on some file systems this modification time check may fail (eg
[Glusterfs #2206](https://github.com/rclone/rclone/issues/2206)) so this
check can be disabled with this flag.`,
			Default:  false,
			Advanced: true,
		}, {
			Name:     "one_file_system",
			Help:     "Don't cross filesystem boundaries (unix/macOS only).",
			Default:  false,
			NoPrefix: true,
			ShortOpt: "x",
			Advanced: true,
		}, {
			Name: "case_sensitive",
			Help: `Force the filesystem to report itself as case sensitive.

Normally the local backend declares itself as case insensitive on
Windows/macOS and case sensitive for everything else.  Use this flag
to override the default choice.`,
			Default:  false,
			Advanced: true,
		}, {
			Name: "case_insensitive",
			Help: `Force the filesystem to report itself as case insensitive

Normally the local backend declares itself as case insensitive on
Windows/macOS and case sensitive for everything else.  Use this flag
to override the default choice.`,
			Default:  false,
			Advanced: true,
			// }, {
			// 	Name:     config.ConfigEncoding,
			// 	Help:     config.ConfigEncodingHelp,
			// 	Advanced: true,
			// 	Default:  defaultEnc,
		}},
	}
	fs.Register(fsi)
}

// Options defines the configuration for this backend
type Options struct {
	FollowSymlinks    bool `config:"copy_links"`
	TranslateSymlinks bool `config:"links"`
	SkipSymlinks      bool `config:"skip_links"`
	NoUTFNorm         bool `config:"no_unicode_normalization"`
	NoCheckUpdated    bool `config:"no_check_updated"`
	NoUNC             bool `config:"nounc"`
	OneFileSystem     bool `config:"one_file_system"`
	CaseSensitive     bool `config:"case_sensitive"`
	CaseInsensitive   bool `config:"case_insensitive"`
	// Enc               encoder.MultiEncoder `config:"encoding"`
}

// Fs represents a local filesystem rooted at root
type Fs struct {
	name string
	root string
	fs   afero.Fs
}

// Object represents a local filesystem object
type Object struct {
	fs   *Fs
	path string
}

// ------------------------------------------------------------

var errLinksAndCopyLinks = errors.New("can't use -l/--links with -L/--copy-links")

// NewFs constructs an Fs from the path
func NewFs(name, root string, m configmap.Mapper) (fs.Fs, error) {
	f := &Fs{
		fs: index.NewFs(name + "@" + root),
	}

	return f, nil
}

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	return f.root
}

// String converts this Fs to a string
func (f *Fs) String() string {
	return fmt.Sprintf("Index file system at %s", f.Root())
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return &fs.Features{}
}

// newObject makes a half completed Object
func (f *Fs) NewObject(ctx context.Context, path string) (fs.Object, error) {
	return &Object{
		fs:   f,
		path: path,
	}, nil
}

// List the objects and directories in dir into entries.  The
// entries can be returned in any order but should be for a
// complete directory.
//
// dir should be "" to list the root, and should not have
// trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
func (f *Fs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
	cwd, err := f.fs.Open(dir)
	if err != nil {
		return nil, err
	}

	defer cwd.Close()

	var entries fs.DirEntries

	files, _ := cwd.Readdir(-1)
	for _, file := range files {
		if file.IsDir() {
			entries = append(entries, fs.NewDir(file.Name(), file.ModTime()))
		} else {
			o, _ := f.NewObject(ctx, file.Name())
			entries = append(entries, o)
		}
	}

	return entries, nil
}

// Put the Object to the local filesystem
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	o, _ := f.NewObject(ctx, src.Remote())
	err := o.Update(ctx, in, src, options...)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// PutStream uploads to the remote path with the modTime given of indeterminate size
func (f *Fs) PutStream(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	return f.Put(ctx, in, src, options...)
}

// Mkdir creates the directory if it doesn't exist
func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	return nil
}

// Rmdir removes the directory
//
// If it isn't empty it will return an error
func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	return nil
}

// Precision of the file system
func (f *Fs) Precision() (precision time.Duration) {
	return time.Second
}

// Move src to this remote using server side move operations.
//
// This is stored with the remote path given
//
// It returns the destination Object and a possible error
//
// Will only be called if src.Fs().Name() == f.Name()
//
// If it isn't possible then return fs.ErrorCantMove
func (f *Fs) Move(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	return nil, nil
}

// DirMove moves src, srcRemote to this remote at dstRemote
// using server side move operations.
//
// Will only be called if src.Fs().Name() == f.Name()
//
// If it isn't possible then return fs.ErrorCantDirMove
//
// If destination exists then return fs.ErrorDirExists
func (f *Fs) DirMove(ctx context.Context, src fs.Fs, srcRemote, dstRemote string) error {
	return nil
}

// Hashes returns the supported hash sets.
func (f *Fs) Hashes() hash.Set {
	return hash.Supported()
}

// ------------------------------------------------------------

// Fs returns the parent Fs
func (o *Object) Fs() fs.Info {
	return o.fs
}

// Return a string version
func (o *Object) String() string {
	return o.path
}

// Remote returns the remote path
func (o *Object) Remote() string {
	return o.path
}

// Hash returns the requested hash of a file as a lowercase hex string
func (o *Object) Hash(ctx context.Context, r hash.Type) (string, error) {
	return "", nil
}

// Size returns the size of an object in bytes
func (o *Object) Size() int64 {
	fi, err := o.fs.fs.Stat(o.path)
	if err != nil {
		return 0
	}

	return fi.Size()
}

// ModTime returns the modification time of the object
func (o *Object) ModTime(ctx context.Context) time.Time {
	fi, _ := o.fs.fs.Stat(o.path)

	return fi.ModTime()
}

// SetModTime sets the modification time of the local fs object
func (o *Object) SetModTime(ctx context.Context, modTime time.Time) error {
	return o.fs.fs.Chtimes(o.path, modTime, modTime)
}

// Storable returns a boolean showing if this object is storable
func (o *Object) Storable() bool {
	return true
}

// Open an object for read
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (in io.ReadCloser, err error) {
	return o.fs.fs.Open(o.path)
}

// Update the object from in with modTime and size
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (err error) {
	fd, err := o.fs.fs.Open(o.path)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err := io.Copy(fd, in); err != nil {
		return err
	}

	return nil
}

// OpenWriterAt opens with a handle for random access writes
//
// Pass in the remote desired and the size if known.
//
// It truncates any existing object
func (f *Fs) OpenWriterAt(ctx context.Context, remote string, size int64) (fs.WriterAtCloser, error) {
	out, err := f.fs.OpenFile(remote, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Remove an object
func (o *Object) Remove(ctx context.Context) error {
	return o.fs.fs.Remove(o.path)
}

// Check the interfaces are satisfied
var (
	_ fs.Fs = &Fs{}
	//	_ fs.Purger         = &Fs{}
	_ fs.PutStreamer    = &Fs{}
	_ fs.Mover          = &Fs{}
	_ fs.DirMover       = &Fs{}
	_ fs.OpenWriterAter = &Fs{}
	_ fs.Object         = &Object{}
)
