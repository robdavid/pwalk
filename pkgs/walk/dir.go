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

type Void struct{}

type DirEntry[T any] interface {
	fs.DirEntry
	GetMapped() T
	Child() *Dir[T]
	WithChild(*Dir[T]) DirEntry[T]
}

type DirEntryFile[T any] struct {
	fs.DirEntry
	mapped T
}

func (f DirEntryFile[T]) GetMapped() T {
	return f.mapped
}

func (DirEntryFile[T]) Child() *Dir[T] {
	return nil
}

func (f DirEntryFile[T]) WithChild(d *Dir[T]) DirEntry[T] {
	panic("Tried to set sub directory " + d.Path.Path() + " on non directory " + f.Name())
}

type DirEntryDir[T any] struct {
	fs.DirEntry
	child  *Dir[T]
	mapped T
}

func MakeDirEntryDir[T any](dirEntry fs.DirEntry, mapped T) DirEntryDir[T] {
	return DirEntryDir[T]{DirEntry: dirEntry, mapped: mapped}
}

func (d DirEntryDir[T]) GetMapped() T {
	return d.mapped
}

func (d DirEntryDir[T]) Child() *Dir[T] {
	return d.child
}

func (d DirEntryDir[T]) WithChild(c *Dir[T]) DirEntry[T] {
	return DirEntryDir[T]{d.DirEntry, c, d.mapped}
}

// Dir contains the results from reading a directory, including
// the directory path, the entries read and any error encountered.
type Dir[T any] struct {
	Path    path.RootedPath
	Entries []DirEntry[T]
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
type MapFn[T any] func(path.RootedPath, os.DirEntry, error) (T, error)

// Read reads the directory at p, and returns a pointer to a [Dir] object
func Read[T any](config *WalkConfig[T], p path.RootedPath) *Dir[T] {
	ents, err := config.Filesystem.ReadDir(p)
	filter := config.Filter
	if filter != nil && err != nil {
		var action FilterAction
		action, err = filter(p, NewAnyDirEntry(config.Filesystem, p), err)
		switch action {
		case FilterSkipDir:
			return &Dir[T]{p, []DirEntry[T]{}, err}
		case FilterSkip:
			return &Dir[T]{p, []DirEntry[T]{}, ErrSkip}
		}
	}
	dirs := make([]DirEntry[T], 0, len(ents))
	for _, ent := range ents {
		if filter != nil {
			var action FilterAction
			action, err := filter(p.Append(ent.Name()), ent, nil)
			switch action {
			case FilterSkipDir:
				return &Dir[T]{p, []DirEntry[T]{}, err}
			case FilterSkip:
				continue
			}
		}
		var mapped T
		if config.Mapper != nil {
			var nerr error
			mapped, nerr = config.Mapper(p, ent, err)
			err = errors.Join(err, nerr)
		}
		if ent.IsDir() {
			dirs = append(dirs, DirEntryDir[T]{ent, nil, mapped})
		} else {
			dirs = append(dirs, DirEntryFile[T]{ent, mapped})
		}
	}
	return &Dir[T]{p, dirs, err}
}

func (d *Dir[T]) EntryIndex(name string) int {
	index, found := slices.BinarySearchFunc(d.Entries, name, func(e DirEntry[T], target string) int {
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

func MakeUnmappedDirEntryDir[T any](ent fs.DirEntry) DirEntry[T] {
	var zero T
	return DirEntryDir[T]{ent, nil, zero}
}

func MakeUnmappedDirEntryFile[T any](ent fs.DirEntry) DirEntry[T] {
	var zero T
	return DirEntryFile[T]{ent, zero}
}

type AnyDirEntry struct {
	RootDirEntry
}

func NewAnyDirEntry(fs Filesystem, p path.RootedPath) *AnyDirEntry {
	return &AnyDirEntry{RootDirEntry: RootDirEntry{filesystem: fs, root: p}}
}

func (r *AnyDirEntry) Name() string {
	if r.root.IsRoot() {
		return ""
	} else {
		return r.root.Top()
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
