package wait

import (
	"sync"
	"time"
)

// WaitGroup is similar with the official sync.WaitGroup, also provide timeout
type WaitGroup struct {
	wg sync.WaitGroup
}

func (w *WaitGroup) Add(delta int) {
	w.wg.Add(delta)
}

func (w *WaitGroup) Done() {
	w.wg.Done()
}

func (w *WaitGroup) Wait() {
	w.wg.Wait()
}

func (w *WaitGroup) WaitWithTimeout(timeout time.Duration) bool {
	c := make(chan struct{})

	go func() {
		defer close(c)
		w.wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}
