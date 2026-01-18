package walk

import "runtime"

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
		config.Filter = f
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
