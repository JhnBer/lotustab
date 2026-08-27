package driver

import (
	"fmt"
	"sync"
)

// Registry is a thread-safe catalogue of database drivers.
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
	order   []string
}

// NewRegistry returns an empty driver registry.
func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]Driver)}
}

// Register adds a driver. IDs must be unique.
func (r *Registry) Register(d Driver) error {
	if d == nil {
		return fmt.Errorf("driver is nil")
	}
	id := d.ID()
	if id == "" {
		return fmt.Errorf("driver id is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.drivers[id]; exists {
		return fmt.Errorf("driver already registered: %s", id)
	}
	r.drivers[id] = d
	r.order = append(r.order, id)
	return nil
}

// Get returns a driver by id.
func (r *Registry) Get(id string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drivers[id]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", id)
	}
	return d, nil
}

// All returns registered drivers in registration order.
func (r *Registry) All() []Driver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Driver, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.drivers[id])
	}
	return out
}

// Infos returns public driver descriptions for the frontend.
func (r *Registry) Infos() []DriverInfo {
	all := r.All()
	out := make([]DriverInfo, 0, len(all))
	for _, d := range all {
		out = append(out, DriverInfo{
			ID:             d.ID(),
			DisplayName:    d.DisplayName(),
			ConnectionSpec: d.ConnectionSpec(),
		})
	}
	return out
}
