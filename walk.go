package pwalk

import (
	"fmt"
	"os"

	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
)

type Workpool interface {
	Run(func())
}

func Walk(p path.RootedPath, fn func(path.RootedPath, dir.DirEntry), wp Workpool) {
	d, err := dir.Read(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		for _, entry := range d.Entries {
			subp := d.Path.Append(entry.Name())
			fn(subp, entry)
			if entry.IsDir() {
				wp.Run(func() {
					Walk(subp, fn, wp)
				})
			}
		}
	}
}
