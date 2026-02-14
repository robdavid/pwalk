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
	for _, onFull := range []workpool.OnFullAction{
		workpool.OnFullInline,
		workpool.OnFullBlock,
		workpool.OnFullBackground,
	} {
		wp := workpool.New(size, 0)
		wp.OnFull = onFull
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
		switch wp.OnFull {
		case workpool.OnFullInline:
			assert.Equal(t, size+1, max, "on full %s", wp.OnFull)
		case workpool.OnFullBackground, workpool.OnFullBlock:
			assert.Equal(t, size, max, "on full %s", wp.OnFull)
		}
	}
}
