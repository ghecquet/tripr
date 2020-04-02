// Package local provides a filesystem interface
package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/lib/file"
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
	fs afero.Fs
	// name        string              // the name of the remote
	// root        string              // The root directory (OS path)
	// opt         Options             // parsed config options
	// features    *fs.Features        // optional features
	// dev         uint64              // device number of root node
	// precisionOk sync.Once           // Whether we need to read the precision
	// precision   time.Duration       // precision of local filesystem
	// warnedMu    sync.Mutex          // used for locking access to 'warned'.
	// warned      map[string]struct{} // whether we have warned about this string

	// // do os.Lstat or os.Stat
	// lstat          func(name string) (os.FileInfo, error)
	// objectHashesMu sync.Mutex // global lock for Object.hashes
}

// Object represents a local filesystem object
type Object struct {
	fs   *Fs // The Fs this object is part of
	file afero.File
	// remote         string // The remote path (encoded path)
	// path           string // The local path (OS path)
	// size           int64  // file metadata - always present
	// mode           os.FileMode
	// modTime        time.Time
	// hashes         map[hash.Type]string // Hashes
	// translatedLink bool                 // Is this object a translated link
}

// ------------------------------------------------------------

var errLinksAndCopyLinks = errors.New("can't use -l/--links with -L/--copy-links")

// NewFs constructs an Fs from the path
func NewFs(name, root string, m configmap.Mapper) (fs.Fs, error) {
	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}
	if opt.TranslateSymlinks && opt.FollowSymlinks {
		return nil, errLinksAndCopyLinks
	}

	if opt.NoUTFNorm {
		fs.Errorf(nil, "The --local-no-unicode-normalization flag is deprecated and will be removed")
	}

	f := &Fs{
		name:   name,
		opt:    *opt,
		warned: make(map[string]struct{}),
		dev:    devUnset,
		lstat:  os.Lstat,
	}
	f.root = cleanRootPath(root, f.opt.NoUNC, f.opt.Enc)
	f.features = (&fs.Features{
		CaseInsensitive:         f.caseInsensitive(),
		CanHaveEmptyDirectories: true,
		IsLocal:                 true,
	}).Fill(f)
	if opt.FollowSymlinks {
		f.lstat = os.Stat
	}

	// Check to see if this points to a file
	fi, err := f.lstat(f.root)
	if err == nil {
		f.dev = readDevice(fi, f.opt.OneFileSystem)
	}
	if err == nil && f.isRegular(fi.Mode()) {
		// It is a file, so use the parent as the root
		f.root = filepath.Dir(f.root)
		// return an error with an fs which points to the parent
		return f, fs.ErrorIsFile
	}
	return f, nil
}

// Determine whether a file is a 'regular' file,
// Symlinks are regular files, only if the TranslateSymlink
// option is in-effect
func (f *Fs) isRegular(mode os.FileMode) bool {
	if !f.opt.TranslateSymlinks {
		return mode.IsRegular()
	}

	// fi.Mode().IsRegular() tests that all mode bits are zero
	// Since symlinks are accepted, test that all other bits are zero,
	// except the symlink bit
	return mode&os.ModeType&^os.ModeSymlink == 0
}

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	return "/"
	// return f.opt.Enc.ToStandardPath(filepath.ToSlash(f.root))
}

// String converts this Fs to a string
func (f *Fs) String() string {
	return fmt.Sprintf("Local file system at %s", f.Root())
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return f.features
}

// caseInsensitive returns whether the remote is case insensitive or not
func (f *Fs) caseInsensitive() bool {
	if f.opt.CaseSensitive {
		return false
	}
	if f.opt.CaseInsensitive {
		return true
	}
	// FIXME not entirely accurate since you can have case
	// sensitive Fses on darwin and case insensitive Fses on linux.
	// Should probably check but that would involve creating a
	// file in the remote to be most accurate which probably isn't
	// desirable.
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

// translateLink checks whether the remote is a translated link
// and returns a new path, removing the suffix as needed,
// It also returns whether this is a translated link at all
//
// for regular files, localPath is returned unchanged
func translateLink(remote, localPath string) (newLocalPath string, isTranslatedLink bool) {
	isTranslatedLink = strings.HasSuffix(remote, linkSuffix)
	newLocalPath = strings.TrimSuffix(localPath, linkSuffix)
	return newLocalPath, isTranslatedLink
}

// newObject makes a half completed Object
func (f *Fs) newObject(remote string) *Object {
	translatedLink := false
	localPath := f.localPath(remote)

	if f.opt.TranslateSymlinks {
		// Possibly receive a new name for localPath
		localPath, translatedLink = translateLink(remote, localPath)
	}

	return &Object{
		fs:             f,
		remote:         remote,
		path:           localPath,
		translatedLink: translatedLink,
	}
}

// Return an Object from a path
//
// May return nil if an error occurred
func (f *Fs) newObjectWithInfo(remote string, info os.FileInfo) (fs.Object, error) {
	o := f.newObject(remote)
	if info != nil {
		o.setMetadata(info)
	} else {
		err := o.lstat()
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fs.ErrorObjectNotFound
			}
			if os.IsPermission(err) {
				return nil, fs.ErrorPermissionDenied
			}
			return nil, err
		}
		// Handle the odd case, that a symlink was specified by name without the link suffix
		if o.fs.opt.TranslateSymlinks && o.mode&os.ModeSymlink != 0 && !o.translatedLink {
			return nil, fs.ErrorObjectNotFound
		}

	}
	if o.mode.IsDir() {
		return nil, errors.Wrapf(fs.ErrorNotAFile, "%q", remote)
	}
	return o, nil
}

// NewObject finds the Object at remote.  If it can't be found
// it returns the error ErrorObjectNotFound.
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	return f.newObjectWithInfo(remote, nil)
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
func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	fd, err := f.fs.Open(dir)
	if err != nil {
		return nil, err
	}

	defer fd.Close()

	for {
		var fis []os.FileInfo
		if useReadDir {
			// Windows and Plan9 read the directory entries with the stat information in which
			// shouldn't fail because of unreadable entries.
			fis, err = fd.Readdir(1024)
			if err == io.EOF && len(fis) == 0 {
				break
			}
		} else {
			// For other OSes we read the names only (which shouldn't fail) then stat the
			// individual ourselves so we can log errors but not fail the directory read.
			var names []string
			names, err = fd.Readdirnames(1024)
			if err == io.EOF && len(names) == 0 {
				break
			}
			if err == nil {
				for _, name := range names {
					namepath := filepath.Join(dir, name)
					fi, err := f.fs.Stat(namepath)
					if err != nil {
						return nil, err
					}
					fis = append(fis, fi)
				}
			}
		}
		if err != nil {
			return nil, err
		}

		for _, fi := range fis {
			name := fi.Name()
			mode := fi.Mode()
			newRemote := path.Join(dir, name)

			// Follow symlinks if required
			if f.opt.FollowSymlinks && (mode&os.ModeSymlink) != 0 {
				localPath := filepath.Join(dir, name)
				fi, err = f.fs.Stat(localPath)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return nil, err
				}

				mode = fi.Mode()
			}

			if fi.IsDir() {
				// Ignore directories which are symlinks.  These are junction points under windows which
				// are kind of a souped up symlink. Unix doesn't have directories which are symlinks.
				// if (mode&os.ModeSymlink) == 0 && f.dev == readDevice(fi, f.opt.OneFileSystem) {
				// 	d := fs.NewDir(newRemote, fi.ModTime())
				// 	entries = append(entries, d)
				// }

				entries = append(entries, fs.NewDir(newRemote, fi.ModTime()))
			} else {
				// Check whether this link should be translated
				if f.opt.TranslateSymlinks && fi.Mode()&os.ModeSymlink != 0 {
					newRemote += linkSuffix
				}
				fso, err := f.newObjectWithInfo(newRemote, fi)
				if err != nil {
					return nil, err
				}
				if fso.Storable() {
					entries = append(entries, fso)
				}
			}
		}
	}
	return entries, nil
}

// Put the Object to the local filesystem
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	// Temporary Object under construction - info filled in by Update()
	o := f.newObject(src.Remote())
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
	return o.file.Name()
}

// Remote returns the remote path
func (o *Object) Remote() string {
	return o.file.Name()
}

// Hash returns the requested hash of a file as a lowercase hex string
func (o *Object) Hash(ctx context.Context, r hash.Type) (string, error) {
	// Check that the underlying file hasn't changed
	oldtime := o.modTime
	oldsize := o.size
	err := o.lstat()
	if err != nil {
		return "", errors.Wrap(err, "hash: failed to stat")
	}

	o.fs.objectHashesMu.Lock()
	hashes := o.hashes
	hashValue, hashFound := o.hashes[r]
	o.fs.objectHashesMu.Unlock()

	if !o.modTime.Equal(oldtime) || oldsize != o.size || hashes == nil || !hashFound {
		var in io.ReadCloser

		if !o.translatedLink {
			var fd *os.File
			fd, err = file.Open(o.path)
			if fd != nil {
				in = newFadviseReadCloser(o, fd, 0, 0)
			}
		} else {
			in, err = o.openTranslatedLink(0, -1)
		}
		if err != nil {
			return "", errors.Wrap(err, "hash: failed to open")
		}
		hashes, err = hash.StreamTypes(in, hash.NewHashSet(r))
		closeErr := in.Close()
		if err != nil {
			return "", errors.Wrap(err, "hash: failed to read")
		}
		if closeErr != nil {
			return "", errors.Wrap(closeErr, "hash: failed to close")
		}
		hashValue = hashes[r]
		o.fs.objectHashesMu.Lock()
		if o.hashes == nil {
			o.hashes = hashes
		} else {
			o.hashes[r] = hashValue
		}
		o.fs.objectHashesMu.Unlock()
	}
	return hashValue, nil
}

// Size returns the size of an object in bytes
func (o *Object) Size() int64 {
	fi, _ := o.file.Stat()

	return fi.Size()
}

// ModTime returns the modification time of the object
func (o *Object) ModTime(ctx context.Context) time.Time {
	fi, _ := o.file.Stat()

	return fi.ModTime()
}

// SetModTime sets the modification time of the local fs object
func (o *Object) SetModTime(ctx context.Context, modTime time.Time) error {
	return o.fs.Chtimes(o.file.Name(), modTime, modTime)
}

// Storable returns a boolean showing if this object is storable
func (o *Object) Storable() bool {
	return true
}

// Open an object for read
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (in io.ReadCloser, err error) {
	return o.fs.Open(o.file.Name())
}

// Update the object from in with modTime and size
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (err error) {

	_, err := io.Copy(out, in)
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}

	// Set the mtime
	err = o.SetModTime(ctx, src.ModTime(ctx))
	if err != nil {
		return err
	}

	// ReRead info now that we have finished
	return o.lstat()
}

// OpenWriterAt opens with a handle for random access writes
//
// Pass in the remote desired and the size if known.
//
// It truncates any existing object
func (f *Fs) OpenWriterAt(ctx context.Context, remote string, size int64) (fs.WriterAtCloser, error) {
	// Temporary Object under construction
	o := f.newObject(remote)

	err := o.mkdirAll()
	if err != nil {
		return nil, err
	}

	if o.translatedLink {
		return nil, errors.New("can't open a symlink for random writing")
	}

	out, err := file.OpenFile(o.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	// Pre-allocate the file for performance reasons
	err = preAllocate(size, out)
	if err != nil {
		fs.Debugf(o, "Failed to pre-allocate: %v", err)
	}
	// Set the file to be a sparse file (important on Windows)
	err = setSparse(out)
	if err != nil {
		fs.Debugf(o, "Failed to set sparse: %v", err)
	}

	return out, nil
}

// Remove an object
func (o *Object) Remove(ctx context.Context) error {
	return remove(o.path)
}

// Check the interfaces are satisfied
var (
	_ fs.Fs             = &Fs{}
	_ fs.Purger         = &Fs{}
	_ fs.PutStreamer    = &Fs{}
	_ fs.Mover          = &Fs{}
	_ fs.DirMover       = &Fs{}
	_ fs.OpenWriterAter = &Fs{}
	_ fs.Object         = &Object{}
)
