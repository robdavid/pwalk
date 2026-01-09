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

func Walk(p path.RootedPath, wp Workpool) {
	dir, err := dir.Read(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		for _, entry := range dir.Entries {
			subp := dir.Path.Append(entry.Name())
			fmt.Println(subp.Path())
			if entry.IsDir() {
				wp.Run(func() {
					Walk(subp, wp)
				})
			}
		}
	}
}
