package pwalk

import (
	"testing"

	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
)

func TestWalk(t *testing.T) {
	wp := workpool.New(12, 32768)
	defer wp.Stop()
	Walk(path.NewAt("/run/user/1000/gvfs/smb-share:server=amycus.snarenet,share=winbackup"), wp)
}
