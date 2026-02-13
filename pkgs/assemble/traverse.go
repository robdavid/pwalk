package assemble

import (
	"github.com/robdavid/pwalk/pkgs/walk"
)

type CoOrd struct {
	Dir   *walk.Dir
	Index int
}

type Location []CoOrd

func (l Location) Last() *CoOrd {
	if len(l) == 0 {
		return nil
	} else {
		return &l[len(l)-1]
	}
}

func (l Location) Push(c CoOrd) Location {
	return append(l, c)
}

func (l Location) Pop() Location {
	return l[:len(l)-1]
}
