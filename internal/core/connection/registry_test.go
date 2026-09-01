package connection

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTrackCancelAll(t *testing.T) {
	r := NewOpRegistry()
	ctxA, doneA := r.Track(context.Background(), "p1")
	defer doneA()
	ctxB, doneB := r.Track(context.Background(), "p1")
	defer doneB()
	ctxOther, doneOther := r.Track(context.Background(), "p2")
	defer doneOther()

	if r.Count("p1") != 2 {
		t.Fatalf("p1 count = %d, want 2", r.Count("p1"))
	}

	r.CancelAll("p1")
	if ctxA.Err() == nil || ctxB.Err() == nil {
		t.Fatal("p1 contexts were not canceled")
	}
	if ctxOther.Err() != nil {
		t.Fatal("p2 context was canceled")
	}
	if r.Count("p1") != 0 {
		t.Fatalf("p1 count after CancelAll = %d", r.Count("p1"))
	}
}

func TestTrackDoneUnregisters(t *testing.T) {
	r := NewOpRegistry()
	_, done := r.Track(context.Background(), "p1")
	done()
	done()
	if r.Count("p1") != 0 {
		t.Fatalf("leaked ops: %d", r.Count("p1"))
	}
	r.CancelAll("p1")
}

func TestCancelEvery(t *testing.T) {
	r := NewOpRegistry()
	ctx1, done1 := r.Track(context.Background(), "a")
	defer done1()
	ctx2, done2 := r.Track(context.Background(), "b")
	defer done2()
	r.CancelEvery()
	if ctx1.Err() == nil || ctx2.Err() == nil {
		t.Fatal("expected every context canceled")
	}
}

func TestTrackParallelCancel(t *testing.T) {
	r := NewOpRegistry()
	const n = 32
	var started, wg sync.WaitGroup
	started.Add(n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, done := r.Track(context.Background(), "p")
			defer done()
			started.Done()
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Error("operation was not canceled")
			}
		}()
	}
	started.Wait()
	r.CancelAll("p")
	wg.Wait()
	if r.Count("p") != 0 {
		t.Fatalf("leaked ops: %d", r.Count("p"))
	}
}
