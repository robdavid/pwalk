package walk

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/robdavid/pwalk/pkgs/path"
)

var ErrSkip = errors.New("entry skipped")

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

// Dir contains the results from reading a directory, including
// the directory path, the entries read and any error encountered.
type Dir struct {
	Path    path.RootedPath
	Entries []DirEntry
	Error   error
}

// FilterAction is returned as the result of the [Filter] function.
// Possible values are [FilterAccept], [FilterSkip] or [FilterSkipDir]
type FilterAction int

const (
	// FilterAccept indicates the file entry is to be included.
	FilterAccept FilterAction = iota
	// FilterSkip indicates the file entry is to be excluded, whether it is a
	// file or a directory.
	FilterSkip
	// FilterSkipDir indicates a directory is to be a excluded. If the entry is
	// a directory, that directory is excluded. Otherwise the parent directory
	// is returned as empty.
	FilterSkipDir
)

type Filter func(path.RootedPath, os.DirEntry, error) (FilterAction, error)

// Read reads the directory at p, and returns a pointer to a [Dir] object
func Read(config *ConfigData, p path.RootedPath) *Dir {
	ents, err := config.Filesystem.ReadDir(p)
	filter := config.Filter
	if filter != nil {
		var action FilterAction
		action, err = filter(p, NewAnyDirEntry(config.Filesystem, p), err)
		switch action {
		case FilterSkipDir:
			return &Dir{p, []DirEntry{}, err}
		case FilterSkip:
			return &Dir{p, []DirEntry{}, ErrSkip}
		}
	}
	dirs := make([]DirEntry, 0, len(ents))
	for _, ent := range ents {
		if filter != nil {
			var action FilterAction
			//p.Push(ent.Name())
			action, err := filter(p.Append(ent.Name()), ent, nil)
			//p.Pop(1)
			switch action {
			case FilterSkipDir:
				return &Dir{p, []DirEntry{}, err}
			case FilterSkip:
				continue
			}
		}
		if ent.IsDir() {
			dirs = append(dirs, DirEntryDir{ent, nil})
		} else {
			dirs = append(dirs, DirEntryFile{ent})
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
	filesystem Filesystem
	root       path.RootedPath
	info       fs.FileInfo
}

func NewRootDirEntry(fs Filesystem, root path.RootedPath) *RootDirEntry {
	return &RootDirEntry{filesystem: fs, root: root}
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
		r.info, err = r.filesystem.Lstat(r.root)
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

func MakeDirEntryDir(ent fs.DirEntry) DirEntry {
	return DirEntryDir{ent, nil}
}

func MakeDirEntryFile(ent fs.DirEntry) DirEntry {
	return DirEntryFile{ent}
}

type AnyDirEntry struct {
	RootDirEntry
}

func NewAnyDirEntry(fs Filesystem, p path.RootedPath) *AnyDirEntry {
	return &AnyDirEntry{RootDirEntry: RootDirEntry{filesystem: fs, root: p}}
}

func (r *AnyDirEntry) Name() string {
	if len(r.root.SubPath) == 0 {
		return ""
	} else {
		return r.root.SubPath[0]
	}
}

// FilterDirSymLinks is a filter function that can be used to skip symbolic links to directories.
// If the entry is a directory and a symbolic link, it will be skipped.
func FilterDirSymLinks(p path.RootedPath, ent os.DirEntry, errIn error) (FilterAction, error) {
	if ent.IsDir() {
		if info, err := ent.Info(); err == nil {
			if info.Mode()&fs.ModeSymlink != 0 {
				return FilterSkip, errIn
			}
		}
	}
	return FilterAccept, errIn
}
