package holdfast

import (
	"context"
	"errors"
	"time"
)

// AcquireRequest は確保リクエストのパラメータ。
// Resource に「何を」、Units に「何個」確保するかを指定する。
type AcquireRequest struct {
	Resource       string
	Units          int
	TTL            time.Duration
	IdempotencyKey string
}

// Lease は確保の結果。TTL 付きで発行される。
type Lease struct {
	ID      string
	Token   uint64    // 単調増加するフェンシングトークン
	Expires time.Time
}

// Receipt は Commit が成功したときの証明。
type Receipt struct {
	LeaseID string
}

// ErrExhausted は確保しようとした Units が枠を超えるときのエラー。
var ErrExhausted = errors.New("holdfast: no units available")

// ErrConflict は古い token や未知の leaseID で Commit/Release したときのエラー。
var ErrConflict = errors.New("holdfast: stale token or unknown lease")

// Store はバックエンドが満たすべき唯一の契約。
// in-memory / Postgres / Raft のどれでも、このメソッドを実装すれば使える。
type Store interface {
	Acquire(ctx context.Context, req AcquireRequest, now time.Time) (Lease, error)
	Commit(ctx context.Context, leaseID string, token uint64) (Receipt, error)
	Release(ctx context.Context, leaseID string, token uint64) error
	Reap(ctx context.Context, now time.Time) (int, error)
}
