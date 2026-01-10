package pwalk

import (
	"fmt"
	"testing"

	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
)

func TestWalk(t *testing.T) {
	wp := workpool.New(12, 12)
	defer wp.Stop()
	Walk(path.NewAt(`w:\`), func(p path.RootedPath, ent dir.DirEntry) { fmt.Println(p.Path()) }, wp)
}
