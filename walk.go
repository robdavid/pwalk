package pwalk

import (
	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
)

type Workpool interface {
	Run(func())
}

func Walk(p path.RootedPath, fn func(dir.Dir, error), wp Workpool) {
	d, err := dir.Read(p)
	fn(d, err)
	for _, entry := range d.Entries {
		if entry.IsDir() {
			wp.Run(func() {
				Walk(d.Path.Append(entry.Name()), fn, wp)
			})
		}
	}
}
