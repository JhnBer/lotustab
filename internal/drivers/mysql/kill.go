package mysql

import (
	"context"
	"sync"
)

// watchKill runs kill if ctx is canceled before stop is called.
// stop is safe to call more than once and prevents a late kill after the
// session has been returned to the pool.
func watchKill(ctx context.Context, kill func()) (stop func()) {
	var (
		mu      sync.Mutex
		stopped bool
		once    sync.Once
		done    = make(chan struct{})
	)
	go func() {
		select {
		case <-ctx.Done():
			mu.Lock()
			defer mu.Unlock()
			if stopped {
				return
			}
			kill()
		case <-done:
		}
	}()
	return func() {
		once.Do(func() {
			mu.Lock()
			stopped = true
			mu.Unlock()
			close(done)
		})
	}
}
