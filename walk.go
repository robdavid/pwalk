package pwalk

import (
	"io/fs"

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

// WalkFn is a function that is called with each path encountered when
// walking the directory tree, along with any error code and directory
// entry details.
type WalkFn = assemble.WalkFn

type MappedWalkFn[T any] = assemble.MappedWalkFn[T]
type Filter = walk.Filter
type MapFn[T any] = walk.MapFn[T]

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

func Walk(root string, fn WalkFn, config ...Config) {
	WalkAndMap(root, nil, toMappedWalkFn(fn), config...)
}

func WalkAndMap[T any](root string, mp MapFn[T], fn MappedWalkFn[T], config ...Config) {
	configData := walk.NewConfigData()
	for _, c := range config {
		c(configData)
	}
	if configData.Workpool == nil {
		wp := workpool.New(configData.Threads, 0)
		defer wp.Stop()
		configData.Workpool = wp
	}
	ass := assemble.New[T](configData, mp, fn)
	defer ass.Close()
	walkConfig := walk.WalkConfig[T]{ConfigData: *configData, Mapper: mp}
	configData.Workpool.Run(func() {
		runWalk(&walkConfig, path.NewAt(root), ass)
	})
	configData.Workpool.Wait()
	ass.Wait()
}
