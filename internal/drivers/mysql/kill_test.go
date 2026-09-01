package mysql

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchKillRunsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var called atomic.Bool
	stop := watchKill(ctx, func() { called.Store(true) })
	defer stop()
	cancel()
	deadline := time.Now().Add(time.Second)
	for !called.Load() {
		if time.Now().After(deadline) {
			t.Fatal("kill was not called")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWatchKillStopPreventsKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var called atomic.Bool
	stop := watchKill(ctx, func() { called.Store(true) })
	stop()
	cancel()
	time.Sleep(30 * time.Millisecond)
	if called.Load() {
		t.Fatal("kill ran after stop")
	}
}

func TestWatchKillStopIdempotent(t *testing.T) {
	stop := watchKill(context.Background(), func() {})
	stop()
	stop()
}
