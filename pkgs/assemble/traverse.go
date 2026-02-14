package assemble

import (
	"github.com/robdavid/pwalk/pkgs/walk"
)

type CoOrd[T any] struct {
	Dir   *walk.Dir[T]
	Index int
}

type Location[T any] []CoOrd[T]

func (l Location[T]) Last() *CoOrd[T] {
	if len(l) == 0 {
		return nil
	} else {
		return &l[len(l)-1]
	}
}

func (l Location[T]) Push(c CoOrd[T]) Location[T] {
	return append(l, c)
}

func (l Location[T]) Pop() Location[T] {
	return l[:len(l)-1]
}
