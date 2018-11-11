package gitfs

import (
	"path/filepath"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

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
	repo := setupTestRepo(t)
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
	repo := setupTestRepo(t)
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
	repo := setupTestRepo(t)
	defer repo.tearDown()

	fs, err := NewROFromHEAD(repo.Repository)
	assert.NoError(t, err)
	file, err := fs.Open("dir1/dir2/test")
	assert.NoError(t, err)
	data, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, "Foobar\n", string(data))
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

func setupTestRepo(t *testing.T) *testRepo {
	name, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	repo, err := git.PlainInit(name, false)
	assert.NoError(t, err)
	wt, err := repo.Worktree()
	assert.NoError(t, err)
	assert.NoError(t, os.MkdirAll(filepath.Join(name, "dir1/dir2"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(name, "dir1/dir3"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(name, "dir1/dir4"), 0755))
	assert.NoError(t, touch(filepath.Join(name, "dir1/dir4/empty")))
	assert.NoError(t, touch(filepath.Join(name, "dir1/dir3/empty")))
	assert.NoError(t, ioutil.WriteFile(filepath.Join(name, "dir1/dir2/test"), []byte("Foobar\n"), 0644))
	assert.NoError(t, wt.AddGlob("*"))
	_, err = wt.Commit("first commit", &git.CommitOptions{
		Author: &object.Signature{
			Email: "me@home",
			Name:  "me",
		},
	})
	assert.NoError(t, err)
	return &testRepo{repo, name}
}

func touch(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	return f.Close()
}
