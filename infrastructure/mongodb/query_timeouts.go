package mongodb

import (
	"context"
	"time"
)

// QueryTimeouts defines standard timeout values for different MongoDB operations.
type QueryTimeouts struct {
	Read  time.Duration
	Write time.Duration
	Index time.Duration
}

// DefaultQueryTimeouts returns standard MongoDB query timeout configuration.
func DefaultQueryTimeouts() QueryTimeouts {
	return QueryTimeouts{
		Read:  10 * time.Second, // Read operations
		Write: 5 * time.Second,  // Write operations (faster, usually simpler)
		Index: 30 * time.Second, // Index creation/management
	}
}

// WithReadTimeout wraps a context with read timeout.
func WithReadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultQueryTimeouts().Read)
}

// WithWriteTimeout wraps a context with write timeout.
func WithWriteTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultQueryTimeouts().Write)
}

// WithIndexTimeout wraps a context with index timeout.
func WithIndexTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultQueryTimeouts().Index)
}
