package workpool

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type Workpool struct {
	request  chan<- func()
	input    <-chan func()
	wg       sync.WaitGroup
	activity sync.WaitGroup
	runLock  sync.Mutex
	Size     int
	Log      *slog.Logger
}

func New(size int, queueSize int) *Workpool {
	ch := make(chan func(), queueSize)
	wp := &Workpool{
		request: ch,
		input:   ch,
		Size:    size,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})),
	}
	for n := range size {
		wp.wg.Go(func() { wp.process(n) })
	}
	return wp
}

func (wp *Workpool) processRecover(n int) {
	if p := recover(); p != nil {
		wp.Log.Error(fmt.Sprintf("%v", p), "worker", n)
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
		wp.Log.Debug("Working running work", "worker", n, "waiting", len(wp.input))
		wp.runFn(fn)
	}
	wp.Log.Debug("Worker exit", "worker", n)
}

func (wp *Workpool) Stop() {
	wp.Log.Debug("Stopping workpool")
	wp.activity.Wait()
	wp.Log.Debug("Work finished")
	close(wp.request)
	wp.wg.Wait()
	wp.Log.Debug("Workers exited")
}

func (wp *Workpool) Run(fn func()) {
	runInline := false
	if cap(wp.request) == 0 {
		wp.Log.Debug("Sending work")
		wp.activity.Add(1)
		wp.request <- fn
		wp.Log.Debug("Sent work")
	} else {
		func() {
			wp.runLock.Lock()
			defer wp.runLock.Unlock()
			wp.Log.Debug("Queuing work", "waiting", len(wp.request))
			if len(wp.request) < cap(wp.request) {
				wp.activity.Add(1)
				wp.request <- fn
				wp.Log.Debug("Queued work", "waiting", len(wp.request))
			} else {
				wp.Log.Debug("Queue full, running inline")
				runInline = true
			}
		}()
	}
	if runInline {
		fn()
	}
}
