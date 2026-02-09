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
	Filter     Filter
	Threads    int
	Workpool   Workpool
	Filesystem Filesystem
}

func NewConfigData() *ConfigData {
	return &ConfigData{
		Filesystem: RealFilesystem{},
		Threads:    runtime.NumCPU(),
	}
}

type Config func(*ConfigData)

func ConfigFilter(f Filter) func(*ConfigData) {
	return func(config *ConfigData) {
		if f == nil {
			return
		}
		if prev := config.Filter; prev == nil {
			config.Filter = f
		} else {
			config.Filter = func(p path.RootedPath, ent os.DirEntry, err error) (FilterAction, error) {
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
