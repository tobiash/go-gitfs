package gitfs

// adapted from go-git

import (
	"path"
	"sort"
	"strings"

	billy "gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/index"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage"
)

// buildTreeHelper converts a given index.Index file into multiple git objects
// reading the blobs from the given filesystem and creating the trees from the
// index structure. The created objects are pushed to a given Storer.
type buildTreeHelper struct {
	fs billy.Filesystem
	s  storage.Storer

	trees   map[string]*object.Tree
	entries map[string]*object.TreeEntry
}

// BuildTree builds the tree objects and push its to the storer, the hash
// of the root tree is returned.
func (h *buildTreeHelper) BuildTree(idx *index.Index) (plumbing.Hash, error) {
	const rootNode = ""
	h.trees = map[string]*object.Tree{rootNode: {}}
	h.entries = map[string]*object.TreeEntry{}

	for _, e := range idx.Entries {
		if err := h.commitIndexEntry(e); err != nil {
			return plumbing.ZeroHash, err
		}
	}

	return h.copyTreeToStorageRecursive(rootNode, h.trees[rootNode])
}

func (h *buildTreeHelper) commitIndexEntry(e *index.Entry) error {
	parts := strings.Split(e.Name, "/")

	var fullpath string
	for _, part := range parts {
		parent := fullpath
		fullpath = path.Join(fullpath, part)

		h.doBuildTree(e, parent, fullpath)
	}

	return nil
}

func (h *buildTreeHelper) doBuildTree(e *index.Entry, parent, fullpath string) {
	if _, ok := h.trees[fullpath]; ok {
		return
	}

	if _, ok := h.entries[fullpath]; ok {
		return
	}

	te := object.TreeEntry{Name: path.Base(fullpath)}

	if fullpath == e.Name {
		te.Mode = e.Mode
		te.Hash = e.Hash
	} else {
		te.Mode = filemode.Dir
		h.trees[fullpath] = &object.Tree{}
	}

	h.trees[parent].Entries = append(h.trees[parent].Entries, te)
}

type sortableEntries []object.TreeEntry

func (sortableEntries) sortName(te object.TreeEntry) string {
	if te.Mode == filemode.Dir {
		return te.Name + "/"
	}
	return te.Name
}
func (se sortableEntries) Len() int               { return len(se) }
func (se sortableEntries) Less(i int, j int) bool { return se.sortName(se[i]) < se.sortName(se[j]) }
func (se sortableEntries) Swap(i int, j int)      { se[i], se[j] = se[j], se[i] }

func (h *buildTreeHelper) copyTreeToStorageRecursive(parent string, t *object.Tree) (plumbing.Hash, error) {
	sort.Sort(sortableEntries(t.Entries))
	for i, e := range t.Entries {
		if e.Mode != filemode.Dir && !e.Hash.IsZero() {
			continue
		}

		path := path.Join(parent, e.Name)

		var err error
		e.Hash, err = h.copyTreeToStorageRecursive(path, h.trees[path])
		if err != nil {
			return plumbing.ZeroHash, err
		}

		t.Entries[i] = e
	}

	o := h.s.NewEncodedObject()
	if err := t.Encode(o); err != nil {
		return plumbing.ZeroHash, err
	}

	return h.s.SetEncodedObject(o)
}
