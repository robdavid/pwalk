package workpool

import (
	"fmt"
	"os"
	"sync"
)

type Workpool struct {
	request  chan<- func()
	input    <-chan func()
	wg       sync.WaitGroup
	activity sync.WaitGroup
	Size     int
}

func New(size int, queueSize int) *Workpool {
	ch := make(chan func(), queueSize)
	wp := &Workpool{
		request: ch,
		input:   ch,
		Size:    size,
	}
	for n := range size {
		wp.wg.Go(func() { wp.process(n) })
	}
	return wp
}

func (wp *Workpool) processRecover(n int) {
	if p := recover(); p != nil {
		fmt.Fprintf(os.Stderr, "worker %d: %v\n", n, p)
		wp.wg.Go(func() { wp.process(n) })
	}
}

func (wp *Workpool) runFn(fn func()) {
	defer wp.activity.Done()
	if fn != nil {
		fn()
	}
}

func (wp *Workpool) process(n int) {
	defer wp.processRecover(n)
	for fn := range wp.input {
		wp.runFn(fn)
	}
	fmt.Printf("worker %d: exit\n", n)
}

func (wp *Workpool) Stop() {
	wp.activity.Wait()
	close(wp.request)
	wp.wg.Wait()
}

func (wp *Workpool) Run(fn func()) {
	wp.activity.Add(1)
	wp.request <- fn
}
