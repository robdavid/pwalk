package walk

import (
	"os"
	"runtime"

	"github.com/robdavid/pwalk/pkgs/path"
)

type Workpool interface {
	Run(fn func())
	Wait()
}

type ConfigData struct {
	Filter     FilterFn
	Threads    int
	Workpool   Workpool
	Filesystem Filesystem
}

type GenConfigData[T any] struct {
	PreProcessor PreProcessFn[T]
}

type WalkConfig[T any] struct {
	ConfigData
	GenConfigData[T]
}

func NewConfigData() *ConfigData {
	return &ConfigData{
		Filesystem: RealFilesystem{},
		Threads:    runtime.NumCPU(),
	}
}

func NewWalkConfig[T any]() *WalkConfig[T] {
	return &WalkConfig[T]{
		ConfigData:    *NewConfigData(),
		GenConfigData: GenConfigData[T]{},
	}
}

// Config is a function that mutates the current walk configuration
type Config func(*ConfigData)

func ConfigFilter(f FilterFn) func(*ConfigData) {
	return func(config *ConfigData) {
		if f == nil {
			return
		}
		if prev := config.Filter; prev == nil {
			config.Filter = f
		} else {
			config.Filter = func(p *path.Path, ent os.DirEntry, err error) (FilterAction, error) {
				if skip, err := prev(p, ent, err); skip == FilterAccept {
					return f(p, ent, err)
				} else {
					return skip, err
				}
			}
		}
	}
}

func ConfigWorkpool(wp Workpool) func(*ConfigData) {
	return func(config *ConfigData) {
		config.Workpool = wp
	}
}

func ConfigThreads(n int) func(*ConfigData) {
	return func(config *ConfigData) {
		config.Threads = n
	}
}

func ConfigFilesystem(fs Filesystem) func(*ConfigData) {
	return func(config *ConfigData) {
		config.Filesystem = fs
	}
}
