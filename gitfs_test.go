package gitfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type testRepo struct {
	*git.Repository
	path string
}

func (r *testRepo) tearDown() {
	os.RemoveAll(r.path)
}

func TestMain(m *testing.M) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
	os.Exit(m.Run())
}

func TestCommitFs(t *testing.T) {
	repo := cloneTestRepo(t, "testdata/a")
	defer repo.tearDown()
	rwFs, err := NewRW(repo.Repository, plumbing.HEAD, func() afero.Fs { return afero.NewMemMapFs() })
	assert.NoError(t, err)
	assert.NoError(t, rwFs.MkdirAll("a/b/c/", 0755))
	assert.NoError(t, afero.Afero{Fs: rwFs}.WriteFile("test", []byte("Hello World"), 0644))
	err = rwFs.Commit("Test commit", git.CommitOptions{
		Author: &object.Signature{
			Email: "me@home",
		},
	})
	assert.NoError(t, err)
}

func TestReadDirNames(t *testing.T) {
	appFs := afero.NewMemMapFs()
	appFs.MkdirAll("src/a/b", 0755)
	appFs.Mkdir("src/b", 0755)
	appFs.Mkdir("src/c", 0755)
	dir, err := appFs.Open("src")
	assert.NoError(t, err)
	dn, err := dir.Readdirnames(1)
	assert.NoError(t, err)
	for i := range dn {
		fmt.Println(dn[i])
	}
	d, err := dir.Readdirnames(1)
	assert.NoError(t, err)
	for i := range d {
		fmt.Println(d[i])
	}
}

func TestReaddir(t *testing.T) {
	repo := cloneTestRepo(t, "testdata/a")
	defer repo.tearDown()

	fs, err := NewROFromHEAD(repo.Repository)
	assert.NoError(t, err)

	dir, err := fs.Open("dir1")
	assert.NoError(t, err)

	dn, err := dir.Readdirnames(1)
	assert.NoError(t, err)
	assert.Len(t, dn, 1)
	assert.ElementsMatch(t, []string{"dir2"}, dn)
	dn, err = dir.Readdirnames(2)
	assert.NoError(t, err)
	assert.Len(t, dn, 2)
	assert.ElementsMatch(t, []string{"dir3", "dir4"}, dn)
	dn, err = dir.Readdirnames(1)
	assert.Equal(t, err, io.EOF)
	assert.Len(t, dn, 0)
}

func TestReadfile(t *testing.T) {
	repo := cloneTestRepo(t, "testdata/a")
	defer repo.tearDown()

	fs, err := NewROFromHEAD(repo.Repository)
	assert.NoError(t, err)
	file, err := fs.Open("dir1/dir2/test")
	assert.NoError(t, err)
	data, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, "Foobar\n", string(data))
}

// var gfs afero.Fs = (*gitfs.GitFS)(nil)

func TestWrite(t *testing.T) {
	repo := initTestRepo(t)
	fmt.Println(repo.path)
	// defer repo.tearDown()
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	wc, err := obj.Writer()
	assert.NoError(t, err)
	_, err = wc.Write([]byte("Hello World"))
	assert.NoError(t, err)
	assert.NoError(t, wc.Close())

	index, err := repo.Storer.Index()
	assert.NoError(t, err)

	entry := index.Add("bla")
	entry.Hash = obj.Hash()
	entry.CreatedAt = time.Now()

	h := &buildTreeHelper{s: repo.Storer}
	tree, err := h.BuildTree(index)
	assert.NoError(t, err)

	commit := &object.Commit{
		Author: object.Signature{
			Email: "me@home",
			Name:  "me",
		},
		Message:  "Test commit",
		TreeHash: tree,
	}

	commitObj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(commitObj); err != nil {
		t.Fatal(err)
	}
	commitHash, _ := repo.Storer.SetEncodedObject(commitObj)

	head, err := repo.Storer.Reference(plumbing.HEAD)
	if err != nil {
		t.Fatal(err)
	}
	name := plumbing.HEAD
	if head.Type() != plumbing.HashReference {
		name = head.Target()
	}
	ref := plumbing.NewHashReference(name, commitHash)
	repo.Storer.SetReference(ref)
}

func cloneTestRepo(t *testing.T, path string) *testRepo {
	name, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainClone(name, true, &git.CloneOptions{
		URL: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &testRepo{repo, name}
}

func initTestRepo(t *testing.T) *testRepo {
	name, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(name, true)
	if err != nil {
		t.Fatal(err)
	}
	return &testRepo{repo, name}
}
