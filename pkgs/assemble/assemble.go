package assemble

import (
	"io/fs"
	"log/slog"
	"os"
	"sync"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
)

type WalkFn = func(path.RootedPath, error, fs.DirEntry)
type Assembly struct {
	root      *walk.Dir
	readState Location
	stream    chan *walk.Dir
	wg        sync.WaitGroup
	WalkFn    WalkFn
	Log       *slog.Logger
}

func New(walkFn WalkFn) *Assembly {
	as := &Assembly{
		WalkFn: walkFn,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelError,
			AddSource: false,
		})),
		stream: make(chan *walk.Dir),
	}
	as.wg.Add(1)
	go as.process()
	return as
}

func (as *Assembly) Add(d *walk.Dir) {
	if len(d.Path.SubPath) == 0 {
		as.root = d
	} else {
		var parent *walk.Dir
		var index int
		next := as.root
		for _, p := range d.Path.SubPath {
			parent = next
			index = parent.EntryIndex(p)
			next = parent.Entries[index].Child()
		}
		parent.Entries[index] = parent.Entries[index].WithChild(d)
		as.Log.Debug("Added directory", "path", d.Path, "entries", len(d.Entries))
	}
}

func (as *Assembly) Next() (fnext path.RootedPath, next walk.DirEntry, direrr error, more bool, blocked bool) {
	if len(as.readState) == 0 {
		as.readState = as.readState.Push(CoOrd{Dir: as.root, Index: 0})
		next = walk.DirEntryDir{DirEntry: walk.NewRootDirEntry(as.root.Path)}
		fnext = as.root.Path
		direrr = as.root.Error
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
					as.readState = as.readState.Push(CoOrd{child, 0})
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

func (as *Assembly) process() {
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
			as.WalkFn(fnext, direrror, next)
		}
	}
}

func (as *Assembly) Wait() {
	as.wg.Wait()
}

func (as *Assembly) Close() {
	close(as.stream)
}

func (as *Assembly) Sink(d *walk.Dir) {
	as.stream <- d
}
