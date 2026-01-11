package pwalk

import (
	"github.com/robdavid/pwalk/pkgs/assemble"
	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
)

type Workpool interface {
	Run(func())
	Wait()
}

func walk(p path.RootedPath, ass *assemble.Assembly, wp Workpool) {
	d := dir.Read(p)
	ass.Sink(d)
	for _, entry := range d.Entries {
		if entry.IsDir() {
			wp.Run(func() {
				walk(d.Path.Append(entry.Name()), ass, wp)
			})
		}
	}
}

func Walk(root string, fn assemble.WalkFn, wp Workpool) {
	ass := assemble.New(fn)
	defer ass.Close()
	wp.Run(func() {
		walk(path.NewAt(root), ass, wp)
	})
	//time.Sleep(1 * time.Second)
	wp.Wait()
	ass.Wait()
}
