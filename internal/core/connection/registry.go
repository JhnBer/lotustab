package connection

import (
	"context"
	"sync"
)

// OpRegistry tracks in-flight operations per profile so Disconnect can cancel them.
type OpRegistry struct {
	mu     sync.Mutex
	nextID uint64
	ops    map[string]map[uint64]context.CancelFunc
}

// NewOpRegistry returns an empty operation registry.
func NewOpRegistry() *OpRegistry {
	return &OpRegistry{ops: make(map[string]map[uint64]context.CancelFunc)}
}

// Track derives a cancellable child of ctx, registers it under profileID, and
// returns a done func that unregisters (and releases) the child. done is safe
// to call more than once.
func (r *OpRegistry) Track(ctx context.Context, profileID string) (context.Context, func()) {
	if r == nil {
		return ctx, func() {}
	}
	ctx, cancel := context.WithCancel(ctx)

	r.mu.Lock()
	id := r.nextID
	r.nextID++
	if r.ops[profileID] == nil {
		r.ops[profileID] = make(map[uint64]context.CancelFunc)
	}
	r.ops[profileID][id] = cancel
	r.mu.Unlock()

	var once sync.Once
	done := func() {
		once.Do(func() {
			cancel()
			r.mu.Lock()
			defer r.mu.Unlock()
			delete(r.ops[profileID], id)
			if len(r.ops[profileID]) == 0 {
				delete(r.ops, profileID)
			}
		})
	}
	return ctx, done
}

// CancelAll cancels every tracked operation for a profile.
func (r *OpRegistry) CancelAll(profileID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, cancel := range r.ops[profileID] {
		cancel()
	}
	delete(r.ops, profileID)
}

// CancelEvery cancels tracked operations for every profile.
func (r *OpRegistry) CancelEvery() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, ops := range r.ops {
		for _, cancel := range ops {
			cancel()
		}
		delete(r.ops, id)
	}
}

// Count returns how many operations are tracked for a profile (tests).
func (r *OpRegistry) Count(profileID string) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.ops[profileID])
}
