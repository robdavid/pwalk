package walk

import (
	"io/fs"
	"os"

	"github.com/robdavid/pwalk/pkgs/path"
)

type Filesystem interface {
	ReadDir(path.RootedPath) ([]fs.DirEntry, error)
	Lstat(path.RootedPath) (fs.FileInfo, error)
}

type RealFilesystem struct{}

func (RealFilesystem) ReadDir(p path.RootedPath) ([]fs.DirEntry, error) {
	return os.ReadDir(p.Path())
}

func (RealFilesystem) Lstat(p path.RootedPath) (fs.FileInfo, error) {
	return os.Lstat(p.Path())
}
