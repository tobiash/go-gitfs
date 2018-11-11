package gitfs

import (
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// OverlayProvider creates a (temporary) filesystem to be used as overlay for the copy-on-write fs
// For example `afero.NewMemMapFs()`
type OverlayProvider = func() afero.Fs

// RWFs is a writable filesystem representation of a git repository at a reference
// Changes will be written to an overlay filesystem and need to be committed explicitly.
type RWFs struct {
	afero.Fs
	base            *Gitfs
	overlay         afero.Fs
	overlayProvider OverlayProvider
	repo            *git.Repository
	ref             plumbing.ReferenceName
}

// NewRW creates a writable filesystem that uses copy-on-write to write changes to the given overlay filesystem
// The overlay filesystem may be requested multiple times, to get clean overlay after committing changes.
func NewRW(repo *git.Repository, ref plumbing.ReferenceName, overlayProvider OverlayProvider) (*RWFs, error) {
	roFs, err := NewROFromRef(repo, ref)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ro fs")
	}
	overlay := overlayProvider()
	fs := afero.NewCopyOnWriteFs(roFs, overlay)
	return &RWFs{fs, roFs, overlay, overlayProvider, repo, ref}, nil
}

// Commit creates a git commit from the filesystem state and moves the reference
// The overlay filesystem is reset to a clean state
func (fs *RWFs) Commit(msg string, opts git.CommitOptions) error {
	hash, err := fs.gitAfero().Tree()
	if err != nil {
		return err
	}
	currentRef, err := resolveRef(fs.repo.Storer, fs.ref)
	if err != nil {
		return err
	}

	commit := &object.Commit{
		Author:       *opts.Author,
		Message:      msg,
		ParentHashes: []plumbing.Hash{currentRef.Hash()},
		TreeHash:     hash,
	}
	commitObj := fs.repo.Storer.NewEncodedObject()
	if err := commit.Encode(commitObj); err != nil {
		return err
	}
	commitHash, err := fs.repo.Storer.SetEncodedObject(commitObj)
	if err != nil {
		return err
	}
	fs.overlay = fs.overlayProvider()
	return fs.updateRef(commitHash)
}

// Stage puts the filesystem state into the index
func (fs *RWFs) Stage() error {
	idx, err := fs.repo.Storer.Index()
	if err != nil {
		return err
	}
	if err = fs.gitAfero().Index(idx); err != nil {
		return err
	}
	if err = fs.repo.Storer.SetIndex(idx); err != nil {
		return err
	}
	fs.overlay = fs.overlayProvider()
	return nil
}

func (fs *RWFs) gitAfero() *GitAfero {
	return &GitAfero{fs: fs, storer: fs.repo.Storer, log: zap.L().Named("gitRWFs")}
}

func (fs *RWFs) updateRef(commitHash plumbing.Hash) error {
	ref, err := fs.repo.Storer.Reference(fs.ref)
	if err != nil {
		return err
	}
	name := fs.ref
	if ref.Type() != plumbing.HashReference {
		name = ref.Target()
	}
	newRef := plumbing.NewHashReference(name, commitHash)
	return fs.repo.Storer.SetReference(newRef)
}
