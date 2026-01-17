package workpool

import (
	"fmt"
	"iter"
	"log/slog"
	"os"
	"sync"
)

type Workpool struct {
	request   chan<- func()
	input     <-chan func()
	wg        sync.WaitGroup
	activity  sync.WaitGroup
	countLock sync.Mutex
	Size      int
	Active    int
	MaxActive int
	Log       *slog.Logger
}

func New(size int, queueSize int) *Workpool {
	ch := make(chan func(), queueSize)
	wp := &Workpool{
		request: ch,
		input:   ch,
		Size:    size,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: false,
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

func (wp *Workpool) addCount(n int) {
	wp.countLock.Lock()
	defer wp.countLock.Unlock()
	wp.Active += n
	if wp.Active > wp.MaxActive {
		wp.MaxActive = wp.Active
	} else if wp.Active < 0 {
		wp.Log.Warn("Correcting bad counter value", "active", wp.Active)
		wp.Active = 0
	}
}

func (wp *Workpool) countRunFn(fn func()) {
	wp.addCount(1)
	defer wp.addCount(-1)
	defer wp.activity.Done()
	if fn != nil {
		fn()
	}
}

func (wp *Workpool) process(n int) {
	defer wp.processRecover(n)
	for fn := range wp.input {
		wp.Log.Debug("Worker running work", "worker", n, "waiting", len(wp.input))
		wp.countRunFn(fn)
	}
	wp.Log.Debug("Worker exit", "worker", n)
}

func (wp *Workpool) Wait() {
	wp.Log.Debug("Waiting for activity to end")
	wp.activity.Wait()
	wp.Log.Debug("Activity ended")
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
	wp.activity.Add(1)
	select {
	case wp.request <- fn:
		wp.Log.Debug("Work queued", "waiting", len(wp.request))
	default:
		wp.Log.Debug("Work running inline", "waiting", len(wp.request))
		wp.runFn(fn)
	}
}

func (wp *Workpool) RunSeq(seq iter.Seq[func()]) {
	for fn := range seq {
		wp.Run(fn)
	}
}
