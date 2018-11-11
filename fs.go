package gitfs

import (
	"os"
	"syscall"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/spf13/afero"
)

func NewROFromHEAD(repo *git.Repository) (*Gitfs, error) {
	return NewROFromRef(repo, plumbing.HEAD)
}

func NewROFromRef(repo *git.Repository, refName plumbing.ReferenceName) (*Gitfs, error) {
	ref, err := resolveRef(repo.Storer, refName)
	if err != nil {
		return nil, err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	return &Gitfs{repo, tree}, nil
}

// Gitfs provides a read-only view to a git tree object
type Gitfs struct {
	repo *git.Repository
	tree *object.Tree
}

func (g *Gitfs) Chmod(s string, f os.FileMode) error {
	return syscall.EPERM
}

func (g *Gitfs) Chtimes(s string, t time.Time, t1 time.Time) error {
	return syscall.EPERM
}

func (g *Gitfs) Create(s string) (afero.File, error) {
	return nil, syscall.EPERM
}

func (g *Gitfs) Mkdir(s string, f os.FileMode) error {
	return syscall.EPERM
}

func (g *Gitfs) MkdirAll(s string, f os.FileMode) error {
	return syscall.EPERM
}

func (g *Gitfs) Name() string {
	return "gitfs"
}

func (g *Gitfs) Open(s string) (afero.File, error) {
	return g.OpenFile(s, os.O_RDONLY, 0)
}

func (g *Gitfs) OpenFile(s string, flag int, f os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	obj, err := g.tree.FindEntry(s)
	if err != nil {
		return nil, &os.PathError{Op: "openFile", Path: s, Err: os.ErrNotExist}
	}
	return &gitfile{
		fs:    g,
		entry: obj,
	}, nil
}

func (g *Gitfs) Remove(s string) error {
	return syscall.EPERM
}

func (g *Gitfs) RemoveAll(s string) error {
	return syscall.EPERM
}

func (g *Gitfs) Rename(s string, s1 string) error {
	return syscall.EPERM
}

func (g *Gitfs) Stat(s string) (os.FileInfo, error) {
	f, err := g.Open(s)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}
