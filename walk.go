package pwalk

import (
	"io/fs"
	"os"

	"github.com/robdavid/pwalk/pkgs/assemble"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
	"github.com/robdavid/pwalk/pkgs/workpool"
)

type Workpool interface {
	Run(func())
	Wait()
}

// Config is a function that mutates the current walk configuration
type Config = walk.Config
type GenConfigData[T any] = walk.GenConfigData[T]

// WalkFn is a function that is called with each path encountered when
// walking the directory tree, along with any error code and directory
// entry details.
type WalkFn = assemble.WalkFn

type MappedWalkFn[T any] = assemble.MappedWalkFn[T]

// FilterFn is the type of a function that can be optionally called with each
// path and directory entry as directories are being read, and can return a
// value to determine whether or not to include a given entry. If an error is
// encountered while reading a directory, the filter will be called with that
// directory path twice. The first time as it is being read from the parent
// directory, and always with an nil error. The second time with the same path
// and the error encountered while reading it. There are three possible return
// values from the filter function:
//   - FilterAccept: The entry is included; normal behavior.
//   - FilterSkip: The entry is not included and it's name will not appear in final results.
//   - FilterSkipDir: Typically the entry's parent directory will be included, but will appear empty,
//     unless the current call is as a result of an error in a directory read (err is non-nil). In
//     this case this directory will be included but will appear empty.
//
// There is also an error return. This error is ultimately passed through to the
// error parameter of the [WalkFn] function.
//
// This function is called concurrently in multiple goroutines as directories
// are being read, so it should be thread-safe.
type FilterFn = walk.FilterFn

type FilterAction = walk.FilterAction

type PreProcessFn[T any] = walk.PreProcessFn[T]

func runWalk[T any](config *walk.WalkConfig[T], p path.RootedPath, ass *assemble.Assembly[T]) {
	d := walk.Read(config, p)
	ass.Sink(d)
	for _, entry := range d.Entries {
		if entry.IsDir() {
			config.Workpool.Run(func() {
				runWalk(config, d.Path.Append(entry.Name()), ass)
			})
		}
	}
}

func toMappedWalkFn(fn WalkFn) MappedWalkFn[walk.Void] {
	return func(p path.RootedPath, ent fs.DirEntry, err error, void walk.Void) {
		fn(p, ent, err)
	}
}

func chainFilter[T any](filter FilterFn, preProcessor PreProcessFn[T]) PreProcessFn[T] {
	if filter == nil {
		return preProcessor
	} else if preProcessor == nil {
		return func(pth path.RootedPath, ent os.DirEntry, err error) (T, FilterAction, error) {
			var zero T
			action, rerr := filter(pth, ent, err)
			return zero, action, rerr
		}
	} else {
		return func(pth path.RootedPath, ent os.DirEntry, err error) (T, FilterAction, error) {
			action, rerr := filter(pth, ent, err)
			if action != walk.FilterAccept {
				var zero T
				return zero, action, rerr
			}
			return preProcessor(pth, ent, err)
		}
	}
}

// Walk traverses the directory tree rooted at the given path and calls fn for each
// file, directory, and error encountered. It walks the tree depth-first in deterministic
// (lexicographical) order.
//
// Parameters:
//   - root: the root directory path to start walking from
//   - fn: WalkFn callback invoked for each path; receives the path, an optional error, and dir entry details.
//     Files are delivered in a consistent depth-first sorted directory entry order.
//   - config: optional variadic configuration functions that can adjust walk behavior.
//
// Configuration functions:
//   - ConfigThreads(n): set the number of worker threads (default: runtime.NumCPU()). This option is
//     ignored if a user supplied workpool is provided.
//   - ConfigWorkpool(wp): provide a custom Workpool implementation (default: internal workpool created).
//   - ConfigFilter(f): supply a Filter function to skip or reject paths during traversal. This is called
//     while directories are being read, so expect it to be called concurrently in multiple goroutines.
//   - ConfigFilesystem(fs): override the default filesystem implementation (for testing or custom storage).
//
// If no config options are provided, Walk uses an internal workpool with thread count
// equal to the system's CPU count.
func Walk(root string, fn WalkFn, config ...Config) {
	GenWalk(root, nil, toMappedWalkFn(fn), config...)
}

// WalkAndMap traverses the directory tree rooted at the given path, optionally
// applies a mapping function to each entry, and then calls fn for each mapped
// result, error, and entry. This is useful for transforming directory entries
// into custom data structures or aggregating information (e.g., computing sums,
// building maps, etc.) during traversal. Note that the mpping function is
// called as the directories are being read, so it is called concurrently in
// multiple goroutines and should be thread-safe.
//
// Parameters:
//   - root: the root directory path to start walking from
//   - fn: MappedWalkFn callback invoked for each entry with the map result; receives the path,
//     the mapped value, an optional error, and dir entry details. Files are delivered in
//     a consistent depth-first sorted directory entry order.
//   - gconf: A struct of walk generic configuration options of type T
//   - config: optional variadic configuration functions that can adjust walk behavior.
//
// Configuration functions for config parameter:
//   - ConfigThreads(n): set the number of worker threads (default: runtime.NumCPU())
//   - ConfigWorkpool(wp): provide a custom Workpool implementation (default: internal workpool created)
//   - ConfigFilter(f): do not use - it will have no effect. Instead use Filter in gconf.
//   - ConfigFilesystem(fs): override the default filesystem implementation (for testing or custom storage)
//
// If no config options are provided, WalkAndMap uses an internal workpool with
// thread count equal to the system's CPU count. The mapping function (if
// provided) is called once per directory and its result is passed to fn along
// with all contained files and subdirectories.
func GenWalk[T any](root string, pre PreProcessFn[T], fn MappedWalkFn[T], config ...Config) {
	configData := walk.NewConfigData()
	for _, c := range config {
		c(configData)
	}
	if configData.Workpool == nil {
		wp := workpool.New(configData.Threads, 0)
		defer wp.Stop()
		configData.Workpool = wp
	}
	walkConfig := walk.WalkConfig[T]{
		ConfigData: *configData,
		GenConfigData: walk.GenConfigData[T]{
			PreProcessor: chainFilter(configData.Filter, pre),
		},
	}

	rootPath := path.NewAt(root)
	rootEnt := walk.NewRootDirEntry(configData.Filesystem, rootPath)

	var mp T
	var action walk.FilterAction
	var err error

	if walkConfig.PreProcessor != nil {
		mp, action, err = walkConfig.PreProcessor(rootPath, rootEnt, nil)
	}
	ass := assemble.New(&walkConfig, mp, fn)
	defer ass.Close()

	switch action {
	case walk.FilterSkip:
		return
	case walk.FilterSkipDir:
		emptyDir := walk.Dir[T]{
			Path:    rootPath,
			Entries: []walk.DirEntry[T]{},
			Error:   err,
		}
		ass.Sink(&emptyDir)
		ass.Wait()
	default:
		configData.Workpool.Run(func() {
			runWalk(&walkConfig, path.NewAt(root), ass)
		})
		configData.Workpool.Wait()
		ass.Wait()
	}
}
