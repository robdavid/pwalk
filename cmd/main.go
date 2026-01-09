package main

import (
	"github.com/robdavid/pwalk"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
)

func main() {
	wp := workpool.New(12, 32768)
	defer wp.Stop()
	pwalk.Walk(path.NewAt("/run/user/1000/gvfs/smb-share:server=amycus.snarenet,share=winbackup"), wp)
}
