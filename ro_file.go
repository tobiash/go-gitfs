package gitfs

import (
	"bytes"
	"errors"
	"io"
	"os"
	"syscall"

	"gopkg.in/src-d/go-git.v4/plumbing/filemode"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type gitfile struct {
	fs           *Gitfs
	entry        *object.TreeEntry
	rdr          io.ReadCloser
	skr          io.ReadSeeker
	readDirCount int
}

func (g *gitfile) Close() error {
	if g.rdr != nil {
		defer func() { g.rdr = nil }()
		return g.rdr.Close()
	}
	return nil
}

func (g *gitfile) Name() string {
	return g.entry.Name
}

func (g *gitfile) initReader() error {
	if g.rdr != nil {
		return nil
	}
	bo, err := g.fs.repo.BlobObject(g.entry.Hash)
	if err != nil {
		return err
	}
	rdr, err := bo.Reader()
	if err != nil {
		return err
	}
	g.rdr = rdr
	return nil
}

func (g *gitfile) initSeeker() error {
	var buf bytes.Buffer
	if err := g.initReader(); err != nil {
		return err
	}
	if g.skr != nil {
		if _, err := io.Copy(&buf, g); err != nil {
			return err
		}
	}
	g.skr = bytes.NewReader(buf.Bytes())
	return nil
}

func (g *gitfile) Read(u []uint8) (int, error) {
	if g.skr != nil {
		return g.skr.Read(u)
	}
	if err := g.initReader(); err != nil {
		return 0, err
	}
	return g.rdr.Read(u)
}

func (g *gitfile) ReadAt(u []uint8, i int64) (int, error) {
	if err := g.initSeeker(); err != nil {
		return 0, err
	}
	_, err := g.skr.Seek(i, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return g.skr.Read(u)
}

func (g *gitfile) patherr(op string, err error) error {
	return &os.PathError{Op: op, Path: g.entry.Name, Err: err}
}

func (g *gitfile) Readdir(n int) ([]os.FileInfo, error) {
	if g.entry.Mode != filemode.Dir {
		return nil, g.patherr("readdir", errors.New("not a directory"))
	}
	tree, err := g.fs.tree.Tree(g.entry.Name)
	if err != nil {
		return nil, g.patherr("readdir", err)
	}
	if n > 0 {
		if len(tree.Entries)-g.readDirCount <= 0 {
			return nil, io.EOF
		}
		var outlen int
		entries := tree.Entries[g.readDirCount:]
		if len(entries) < n {
			outlen = len(entries)
		} else {
			outlen = n
		}
		res := make([]os.FileInfo, outlen)
		for i := range res {
			res[i] = &fileinfo{&entries[i]}
		}
		g.readDirCount += outlen
		return res, nil
	}
	return nil, nil
}

func (g *gitfile) Readdirnames(i int) ([]string, error) {
	fi, err := g.Readdir(i)
	res := make([]string, len(fi))
	for idx := range fi {
		res[idx] = fi[idx].Name()
	}
	return res, err
}

func (g *gitfile) Seek(i int64, i1 int) (int64, error) {
	return g.skr.Seek(i, i1)
}

func (g *gitfile) Stat() (os.FileInfo, error) {
	return &fileinfo{g.entry}, nil
}

func (g *gitfile) Sync() error {
	return nil
}

func (g *gitfile) Truncate(i int64) error {
	return syscall.EPERM
}

func (g *gitfile) Write(u []uint8) (int, error) {
	return 0, syscall.EPERM
}

func (g *gitfile) WriteAt(u []uint8, i int64) (int, error) {
	return 0, syscall.EPERM
}

func (g *gitfile) WriteString(s string) (int, error) {
	return 0, syscall.EPERM
}
