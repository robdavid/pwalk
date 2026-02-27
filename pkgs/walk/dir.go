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

var Nil Void = Void{}

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
	Path    path.Path
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

type PreProcessFn[T any] = func(path.Path, os.DirEntry, error) (T, FilterAction, error)
type FilterFn = func(path.Path, os.DirEntry, error) (FilterAction, error)

// Read reads the directory at p, and returns a pointer to a [Dir] object
func Read[T any](config *WalkConfig[T], p path.Path) *Dir[T] {
	ents, err := config.Filesystem.ReadDir(p)
	preProcessor := config.PreProcessor
	if preProcessor != nil && err != nil {
		var action FilterAction
		_, action, err = preProcessor(p, NewAnyDirEntry(config.Filesystem, p), err)
		switch action {
		case FilterSkipDir:
			return &Dir[T]{p, []DirEntry[T]{}, err}
		case FilterSkip:
			return &Dir[T]{p, []DirEntry[T]{}, ErrSkip}
		}
	}
	dirs := make([]DirEntry[T], 0, len(ents))
	for _, ent := range ents {
		var mapped T
		if preProcessor != nil {
			var action FilterAction
			var nerr error
			mapped, action, nerr = preProcessor(p.Append(ent.Name()), ent, err)
			err = errors.Join(err, nerr)
			switch action {
			case FilterSkipDir:
				return &Dir[T]{p, []DirEntry[T]{}, err}
			case FilterSkip:
				continue
			}
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
	root       path.Path
	info       fs.FileInfo
}

func NewRootDirEntry(fs Filesystem, root path.Path) *RootDirEntry {
	return &RootDirEntry{filesystem: fs, root: root}
}

func (r *RootDirEntry) Name() string {
	return ""
}

func (r *RootDirEntry) IsDir() bool {
	return true
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

func NewAnyDirEntry(fs Filesystem, p path.Path) *AnyDirEntry {
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
// If the entry is a directory and a symbolic link, it will be skipped, unless it is the root.
func FilterDirSymLinks(p path.Path, ent os.DirEntry, errIn error) (FilterAction, error) {
	if ent.IsDir() && !p.IsRoot() {
		if info, err := ent.Info(); err == nil {
			if info.Mode()&fs.ModeSymlink != 0 {
				return FilterSkip, errIn
			}
		}
	}
	return FilterAccept, errIn
}

func ChainPreprocessors[T any](fns ...PreProcessFn[T]) PreProcessFn[T] {
	return func(p path.Path, ent os.DirEntry, errIn error) (value T, action FilterAction, err error) {
		err = errIn
		for _, fn := range fns {
			value, action, err = fn(p, ent, err)
			if action != FilterAccept {
				break
			}
		}
		return
	}
}
