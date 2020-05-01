package aferofs

import (
	"errors"
	"os"
	"strings"
	"time"

	_ "github.com/ghecquet/tripr/poc/cells/client/resolver"
	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4"
)

var ErrCrossRepository = errors.New("cross-repository operation not allowed")

var _ afero.Lstater = (*GitDirFs)(nil)

type GitDirFs struct {
	worktree afero.Fs
	git      afero.Fs
}

// NewGitDirFs groups a worktree and a git dir from different locations
func NewGitDirFs(worktree afero.Fs, git afero.Fs) afero.Fs {
	return &GitDirFs{
		worktree: worktree,
		git:      git,
	}
}

// func (f *GitDirFs) ReadDir(name string) ([]os.FileInfo, error) {
// 	if belongsToGitDir(name) {
// 		return f.git.ReadDir(name)
// 	}

// 	return f.worktree.ReadDir(name)
// }

func (f *GitDirFs) Chtimes(name string, added, modified time.Time) error {
	if belongsToGitDir(name) {
		return f.git.Chtimes(name, added, modified)
	}

	return f.worktree.Chtimes(name, added, modified)
}

func (f *GitDirFs) Chmod(name string, mode os.FileMode) error {
	if belongsToGitDir(name) {
		return f.git.Chmod(name, mode)
	}

	return f.worktree.Chmod(name, mode)
}

func (f *GitDirFs) Name() string {
	return "GitDirFs"
}

func (f *GitDirFs) Stat(name string) (os.FileInfo, error) {
	if belongsToGitDir(name) {
		return f.git.Stat(name)
	}

	return f.worktree.Stat(name)
}

func (f *GitDirFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	if belongsToGitDir(name) {
		return f.git.(afero.Lstater).LstatIfPossible(name)
	}

	return f.worktree.(afero.Lstater).LstatIfPossible(name)
}

func (f *GitDirFs) Rename(oldName, newName string) error {
	if belongsToGitDir(oldName) {
		if !belongsToGitDir(newName) {
			return ErrCrossRepository
		}
		return f.git.Rename(oldName, newName)
	}

	if belongsToGitDir(newName) {
		return ErrCrossRepository
	}

	return f.worktree.Rename(oldName, newName)
}

func (f *GitDirFs) RemoveAll(path string) error {
	if belongsToGitDir(path) {
		return f.git.RemoveAll(path)
	}

	return f.worktree.RemoveAll(path)
}

func (f *GitDirFs) Remove(name string) error {
	if belongsToGitDir(name) {
		return f.git.Remove(name)
	}

	return f.worktree.Remove(name)
}

func (f *GitDirFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if belongsToGitDir(name) {
		return f.git.OpenFile(name, flag, perm)
	}

	return f.worktree.OpenFile(name, flag, perm)
}

func (f *GitDirFs) Open(name string) (afero.File, error) {
	if belongsToGitDir(name) {
		return f.git.Open(name)
	}

	return f.worktree.Open(name)
}

func (f *GitDirFs) Mkdir(name string, perm os.FileMode) error {
	if belongsToGitDir(name) {
		return f.git.Mkdir(name, perm)
	}

	return f.worktree.Mkdir(name, perm)
}

func (f *GitDirFs) MkdirAll(path string, perm os.FileMode) error {
	if belongsToGitDir(path) {
		return f.git.MkdirAll(path, perm)
	}

	return f.worktree.MkdirAll(path, perm)
}

func (f *GitDirFs) Create(name string) (afero.File, error) {
	if belongsToGitDir(name) {
		return f.git.Create(name)
	}

	return f.worktree.Create(name)
}

// Utils
func belongsToGitDir(name string) bool {
	return strings.HasPrefix(name, "/"+git.GitDirName)
}
