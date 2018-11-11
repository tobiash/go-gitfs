package gitfs

import (
	"encoding/hex"
	"io"
	"os"
	"path"
	"sort"

	"github.com/spf13/afero"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/index"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

// GitAfero is a helper to use an afero Fs in place of a git working directory.
// It has tools to write the filesystem to a git repository index
type GitAfero struct {
	fs     afero.Fs
	storer storage.Storer
	log    *zap.Logger
}

// Tree creates a tree object from the contents of the wrapped filesystem
func (g *GitAfero) Tree() (plumbing.Hash, error) {
	return g.buildTreeRecursive("")
}

// Index writes the contents of the filesystem to the index
func (g *GitAfero) Index(idx *index.Index) error {
	return nil
}

// Store stores a single file as git blob object and returns the hash
func (g *GitAfero) Store(path string, fi os.FileInfo) (hash plumbing.Hash, err error) {
	// TODO Symlink support
	obj := g.storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(fi.Size())
	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	defer ioutil.CheckClose(w, &err)
	f, err := g.fs.Open(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	defer f.Close()
	if _, err = io.Copy(w, f); err != nil {
		return plumbing.ZeroHash, err
	}
	if err = w.Close(); err != nil {
		return plumbing.ZeroHash, err
	}
	return g.storer.SetEncodedObject(obj)
}

func (g *GitAfero) indexDir(idx *index.Index, directory string) error {
	fis, err := (&afero.Afero{Fs: g.fs}).ReadDir(directory)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		name := path.Join(directory, fi.Name())
		if fi.IsDir() {
			err = g.indexDir(idx, name)
		} else {
			err = g.indexFile(idx, name, fi)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GitAfero) indexFile(idx *index.Index, filename string, fi os.FileInfo) error {
	hash, err := g.Store(filename, fi)
	if err != nil {
		return err
	}
	entry, err := idx.Entry(filename)
	if err != nil && err != index.ErrEntryNotFound {
		return err
	}
	if err == index.ErrEntryNotFound {
		entry = idx.Add(filename)
	}
	return g.updateEntry(entry, filename, fi, hash)
}

func (g *GitAfero) updateEntry(e *index.Entry, filename string, fi os.FileInfo, h plumbing.Hash) (err error) {
	e.Hash = h
	e.ModifiedAt = fi.ModTime()
	e.Mode, err = filemode.NewFromOSFileMode(fi.Mode())
	if err != nil {
		return err
	}
	if e.Mode.IsRegular() {
		e.Size = uint32(fi.Size())
	}
	// TODO System info?
	return nil
}
func (g *GitAfero) buildTreeRecursive(fullPath string) (plumbing.Hash, error) {
	tree := &object.Tree{}
	fis, err := afero.Afero{Fs: g.fs}.ReadDir(fullPath)
	sort.Sort(sortableFileInfo(fis))
	if err != nil {
		return plumbing.ZeroHash, err
	}
	tree.Entries = make([]object.TreeEntry, len(fis))
	for idx, child := range fis {
		var hash plumbing.Hash
		var mode filemode.FileMode

		if child.IsDir() {
			mode = filemode.Dir
			hash, err = g.buildTreeRecursive(path.Join(fullPath, child.Name()))
		} else {
			mode, err = filemode.NewFromOSFileMode(child.Mode())
			if err != nil {
				return plumbing.ZeroHash, err
			}
			hash, err = g.Store(path.Join(fullPath, child.Name()), child)
		}
		if err != nil {
			return plumbing.ZeroHash, err
		}
		g.log.Debug("add tree entry",
			zap.String("path", fullPath),
			zap.String("hash", hex.EncodeToString(hash[:])),
		)
		tree.Entries[idx] = object.TreeEntry{
			Hash: hash,
			Name: child.Name(),
			Mode: mode,
		}
	}
	o := g.storer.NewEncodedObject()
	if err = tree.Encode(o); err != nil {
		return plumbing.ZeroHash, err
	}
	return g.storer.SetEncodedObject(o)
}

type sortableFileInfo []os.FileInfo

func (se sortableFileInfo) sortName(i int) string {
	fi := se[i]
	if fi.IsDir() {
		return fi.Name() + "/"
	}
	return fi.Name()
}

func (se sortableFileInfo) Len() int           { return len(se) }
func (se sortableFileInfo) Less(i, j int) bool { return se.sortName(i) < se.sortName(j) }
func (se sortableFileInfo) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
