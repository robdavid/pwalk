package dir

import (
	"io/fs"
	"os"

	"github.com/robdavid/pwalk/pkgs/path"
)

type DirEntry interface {
	fs.DirEntry
	Child() *Dir
	WithChild(*Dir) DirEntry
}

type DirEntryFile struct {
	fs.DirEntry
}

func (DirEntryFile) Child() *Dir {
	return nil
}

func (f DirEntryFile) WithChild(d *Dir) DirEntry {
	panic("Tried to set sub directory " + d.Path.Path() + " on non directory " + f.Name())
}

type DirEntryDir struct {
	fs.DirEntry
	child *Dir
}

func (d DirEntryDir) Child() *Dir {
	return d.child
}

func (d DirEntryDir) WithChild(c *Dir) DirEntry {
	return DirEntryDir{d.DirEntry, c}
}

type Dir struct {
	Path    path.RootedPath
	Entries []DirEntry
}

func Read(p path.RootedPath) (Dir, error) {
	ents, err := os.ReadDir(p.Path())
	dirs := make([]DirEntry, len(ents))
	for i, ent := range ents {
		if ent.IsDir() {
			dirs[i] = DirEntryDir{ent, nil}
		} else {
			dirs[i] = DirEntryFile{ent}
		}
	}
	return Dir{p, dirs}, err
}
