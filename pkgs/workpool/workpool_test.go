package workpool_test

import (
	"sync"
	"testing"
	"time"

	"github.com/robdavid/pwalk/pkgs/workpool"
	"github.com/stretchr/testify/assert"
)

func TestRunPool(t *testing.T) {
	var lock sync.Mutex
	const size = 10
	wp := workpool.New(size, 0)
	count := 0
	max := 0
	inc := func() {
		lock.Lock()
		defer lock.Unlock()
		count++
		if count > max {
			max = count
		}
	}
	dec := func() {
		lock.Lock()
		defer lock.Unlock()
		count--
	}
	incdec := func() {
		inc()
		time.Sleep(100 * time.Millisecond)
		dec()
	}
	for range size * 3 {
		wp.Run(incdec)
	}
	wp.Stop()
	assert.Equal(t, size+1, max)
}
