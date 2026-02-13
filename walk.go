package pwalk

import (
	"github.com/robdavid/pwalk/pkgs/assemble"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
	"github.com/robdavid/pwalk/pkgs/workpool"
)

type Workpool interface {
	Run(func())
	Wait()
}

type Config = walk.Config
type WalkFn = assemble.WalkFn

func runWalk(config *walk.ConfigData, p path.RootedPath, ass *assemble.Assembly) {
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

func Walk(root string, fn WalkFn, config ...Config) {
	configData := walk.NewConfigData()
	for _, c := range config {
		c(configData)
	}
	if configData.Workpool == nil {
		wp := workpool.New(configData.Threads, 0)
		defer wp.Stop()
		configData.Workpool = wp
	}
	ass := assemble.New(configData, fn)
	defer ass.Close()
	configData.Workpool.Run(func() {
		runWalk(configData, path.NewAt(root), ass)
	})
	//time.Sleep(1 * time.Second)
	configData.Workpool.Wait()
	ass.Wait()
}
