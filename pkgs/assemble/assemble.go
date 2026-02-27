package assemble

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sync"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
)

// WalkFn is a function that is called with each path encountered when
// walking the directory tree, along with any error code and directory
// entry details.
type WalkFn = func(path.Path, fs.DirEntry, error)
type MappedWalkFn[T any] = func(path.Path, fs.DirEntry, error, T)

type Assembly[T any] struct {
	root       *walk.Dir[T]
	rootMapped T
	readState  Location[T]
	stream     chan *walk.Dir[T]
	wg         sync.WaitGroup
	config     *walk.ConfigData
	WalkFn     MappedWalkFn[T]
	Log        *slog.Logger
}

func New[T any](config *walk.WalkConfig[T], rootMapped T, walkFn MappedWalkFn[T]) *Assembly[T] {
	as := &Assembly[T]{
		WalkFn: walkFn,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelError,
			AddSource: false,
		})),
		rootMapped: rootMapped,
		config:     &config.ConfigData,
		stream:     make(chan *walk.Dir[T]),
	}
	as.wg.Add(1)
	go as.process()
	return as
}

func (as *Assembly[T]) Add(d *walk.Dir[T]) {
	if d.Path.IsRoot() {
		as.root = d
	} else {
		var parent *walk.Dir[T]
		var index int
		next := as.root
		for p := range d.Path.SubPaths() {
			parent = next
			index = parent.EntryIndex(p)
			if index < 0 {
				panic(fmt.Errorf("Cannot find %s in %s", p, parent.Path))
			}
			next = parent.Entries[index].Child()
		}
		parent.Entries[index] = parent.Entries[index].WithChild(d)
		as.Log.Debug("Added directory", "path", d.Path, "entries", len(d.Entries))
	}
}

// Next returns the next entry in the sorted sequence of directory entries.
// Returned are the path and the directory entry, along with any associated
// error. Also returned are two flags.
//   - more - If false then there are no further entries. No entry data is returned.
//   - blocked - If true then the next entry has yet to be inserted into the [Assembly]. Subsequent calls to [Assembly.Add]
//     may then provide the required entry in which case the flag will be cleared on the next call to this method.
func (as *Assembly[T]) Next() (fnext path.Path, next walk.DirEntry[T], direrr error, more bool, blocked bool) {
	if len(as.readState) == 0 {
		as.readState = as.readState.Push(CoOrd[T]{Dir: as.root, Index: 0})
		dirent := walk.NewRootDirEntry(as.config.Filesystem, as.root.Path)
		next = walk.MakeDirEntryDir(dirent, as.rootMapped)
		fnext = as.root.Path
		direrr = errors.Join(direrr, as.root.Error)
		more = true
		return
	}
	for {
		if len(as.readState) == 0 {
			blocked = true
			return
		}
		current := as.readState.Last()
		if current.Index >= len(current.Dir.Entries) {
			// Finished directory, resuming in parent
			as.readState = as.readState.Pop()
			parent := as.readState.Last()
			if parent != nil {
				// Dropped ref to finished directory
				parent.Dir.Entries[parent.Index] = parent.Dir.Entries[parent.Index].WithChild(nil)
				parent.Index++
			}
		} else {
			more = true
			next = current.Dir.Entries[current.Index]
			fnext = current.Dir.Path.Append(next.Name())
			if next.IsDir() {
				if child := next.Child(); child == nil {
					blocked = true
					as.Log.Debug("Waiting for directory", "path", fnext)
				} else {
					direrr = child.Error
					if direrr == walk.ErrSkip {
						current.Index++
						direrr = nil
						continue
					}
					as.readState = as.readState.Push(CoOrd[T]{child, 0})
					as.Log.Debug("Found directory", "path", fnext)
				}
			} else {
				current.Index++
				as.Log.Debug("Found file", "path", fnext)
			}
			return
		}
	}
}

func (as *Assembly[T]) process() {
	defer as.wg.Done()
	for d := range as.stream {
		as.Add(d)
		for {
			fnext, next, direrror, more, blocked := as.Next()
			if !more {
				return
			} else if blocked {
				break
			}
			as.WalkFn(fnext, next, direrror, next.GetMapped())
		}
	}
}

func (as *Assembly[T]) Wait() {
	as.wg.Wait()
}

func (as *Assembly[T]) Close() {
	close(as.stream)
}

func (as *Assembly[T]) Sink(d *walk.Dir[T]) {
	as.stream <- d
}
