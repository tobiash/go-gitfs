package gitfs

import (
	"errors"
	"os"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing/filemode"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type fileinfo struct{ *object.TreeEntry }

func (f *fileinfo) IsDir() bool {
	return f.TreeEntry.Mode == filemode.Dir
}

func (f *fileinfo) ModTime() time.Time {
	panic(errors.New("*fileinfo.ModTime not implemented"))
}

func (f *fileinfo) Mode() os.FileMode {
	fm, _ := f.TreeEntry.Mode.ToOSFileMode()
	return fm
}

func (f *fileinfo) Name() string {
	return f.TreeEntry.Name
}

func (f *fileinfo) Size() int64 {
	panic(errors.New("*fileinfo.Size not implemented"))
}

func (f *fileinfo) Sys() interface{} {
	panic(errors.New("*fileinfo.Sys not implemented"))
}
