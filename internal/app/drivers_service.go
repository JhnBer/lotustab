package app

import "lotustab/internal/core/driver"

// DriversService exposes registered database drivers to the frontend.
type DriversService struct {
	registry *driver.Registry
}

// NewDriversService constructs the Wails drivers service.
func NewDriversService(registry *driver.Registry) *DriversService {
	return &DriversService{registry: registry}
}

// List returns driver ids, labels, and connection form specs.
func (s *DriversService) List() []driver.DriverInfo {
	return s.registry.Infos()
}
