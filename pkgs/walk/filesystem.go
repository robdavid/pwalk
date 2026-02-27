package walk

import (
	"io/fs"
	"os"

	"github.com/robdavid/pwalk/pkgs/path"
)

type Filesystem interface {
	ReadDir(*path.Path) ([]fs.DirEntry, error)
	Lstat(*path.Path) (fs.FileInfo, error)
}

type RealFilesystem struct{}

func (RealFilesystem) ReadDir(p *path.Path) ([]fs.DirEntry, error) {
	return os.ReadDir(p.Path())
}

func (RealFilesystem) Lstat(p *path.Path) (fs.FileInfo, error) {
	return os.Lstat(p.Path())
}
