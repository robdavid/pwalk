package dir

import (
	"io/fs"
	"os"
	"slices"
	"strings"

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
	Error   error
}

func Read(p path.RootedPath) *Dir {
	ents, err := os.ReadDir(p.Path())
	dirs := make([]DirEntry, len(ents))
	for i, ent := range ents {
		if ent.IsDir() {
			dirs[i] = DirEntryDir{ent, nil}
		} else {
			dirs[i] = DirEntryFile{ent}
		}
	}
	return &Dir{p, dirs, err}
}

func (d *Dir) EntryIndex(name string) int {
	index, found := slices.BinarySearchFunc(d.Entries, name, func(e DirEntry, target string) int {
		return strings.Compare(e.Name(), target)
	})
	if !found {
		return -1
	} else {
		return index
	}
}

type RootDirEntry struct {
	root path.RootedPath
	info fs.FileInfo
}

func NewRootDirEntry(root path.RootedPath) *RootDirEntry {
	return &RootDirEntry{root: root}
}

func (r *RootDirEntry) Name() string {
	return ""
}

func (r *RootDirEntry) IsDir() bool {
	return r.Type().IsDir()
}

func (r *RootDirEntry) Info() (fs.FileInfo, error) {
	if r.info == nil {
		var err error
		r.info, err = os.Stat(r.root.Path())
		return r.info, err
	} else {
		return r.info, nil
	}
}

func (r *RootDirEntry) Type() fs.FileMode {
	if info, err := r.Info(); err != nil {
		return fs.FileMode(fs.ModeDir | 0o555)
	} else {
		return info.Mode()
	}
}

func NewDirEntryDir(ent fs.DirEntry) DirEntry {
	return DirEntryDir{ent, nil}
}
