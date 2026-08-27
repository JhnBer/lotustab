package driver

import "context"

// Driver is the port every database engine must implement.
type Driver interface {
	ID() string
	DisplayName() string
	ConnectionSpec() []Field
	Open(ctx context.Context, cfg ConnConfig) (Conn, error)
}
