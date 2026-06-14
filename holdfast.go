package holdfast

import (
	"context"
	"errors"
	"time"
)

type RegisterRequest struct {
	Resource string
	Capacity int
}

type AcquireRequest struct {
	Resource       string
	Units          int
	TTL            time.Duration
	IdempotencyKey string
}

type CommitRequest struct {
	LeaseID string
	Token   uint64
}

type ReleaseRequest struct {
	LeaseID string
	Token   uint64
}

type Lease struct {
	ID      string
	Token   uint64
	Expires time.Time
}

type Receipt struct {
	LeaseID string
}

var ErrUnknownResource = errors.New("holdfast: unknown resource")

var ErrExhausted = errors.New("holdfast: no units available")

var ErrConflict = errors.New("holdfast: stale token or unknown lease")

type Store interface {
	Register(ctx context.Context, req RegisterRequest, now time.Time) error
	Acquire(ctx context.Context, req AcquireRequest, now time.Time) (Lease, error)
	Commit(ctx context.Context, req CommitRequest) (Receipt, error)
	Release(ctx context.Context, req ReleaseRequest) error
	Reap(ctx context.Context, now time.Time) (int, error)
}
