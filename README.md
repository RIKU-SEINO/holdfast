# holdfast

[![Go Reference](https://pkg.go.dev/badge/github.com/RIKU-SEINO/holdfast.svg)](https://pkg.go.dev/github.com/RIKU-SEINO/holdfast)
[![Go Report Card](https://goreportcard.com/badge/github.com/RIKU-SEINO/holdfast)](https://goreportcard.com/report/github.com/RIKU-SEINO/holdfast)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**A domain-agnostic lease primitive for Go.**

Reserve N units of any resource with a TTL and a fencing token. Commit or release when done. Works the same whether you're building seat reservations, inventory holds, rate limiters, or distributed mutexes.

```go
lease, err := store.Acquire(ctx, holdfast.AcquireRequest{
    Resource:       "seats:row-A",
    Units:          2,
    TTL:            5 * time.Minute,
    IdempotencyKey: "order-789",
}, time.Now())

// lease.Token is a monotonically increasing fencing token
store.Commit(ctx, holdfast.CommitRequest{LeaseID: lease.ID, Token: lease.Token})
```

---

## Requirements

- Go 1.26+

---

## Why holdfast?

The same pattern appears across domains:

| Use case | Resource | Units |
|---|---|---|
| Seat reservation | `concert:row-A` | 2 |
| Inventory hold | `sku:abc123` | 10 |
| Rate limiting | `api:user:42` | 1 |
| Distributed mutex | `job:nightly-sync` | 1 |
| Connection pool | `db:primary` | 5 |

`Units=1` gives you a distributed mutex. `Units=N` gives you a distributed semaphore. holdfast knows nothing about seats or inventory — it only knows: **reserve N units with a TTL, hand back a fencing token**.

---

## How it works

### Architecture

```
┌─────────────────────────────────────────────┐
│  Your application                           │
│  store.Acquire / Commit / Release / Reap    │
└────────────────────┬────────────────────────┘
                     │ holdfast.Store interface
        ┌────────────┼─────────────┐
        ▼            ▼             ▼
   ┌─────────┐  ┌──────────┐  ┌────────┐
   │ memory  │  │ postgres │  │  raft  │
   │  (dev)  │  │ (single  │  │  (HA,  │
   │         │  │ primary) │  │ multi- │
   │         │  │          │  │  node) │
   └─────────┘  └──────────┘  └────────┘
```

Correctness reduces entirely to **the atomicity of the chosen `Store`**. Swap backends without changing application code.

### The `Store` interface

```go
type Store interface {
    Register(ctx context.Context, req RegisterRequest, now time.Time) error
    Acquire(ctx context.Context, req AcquireRequest, now time.Time) (Lease, error)
    Commit(ctx context.Context, req CommitRequest) (Receipt, error)
    Release(ctx context.Context, req ReleaseRequest) error
    Reap(ctx context.Context, now time.Time) (int, error)
}
```

### Fencing tokens

Every lease carries a monotonically increasing `Token`. Pass it to your downstream resource to reject stale writes from paused clients — even after GC pauses, network partitions, or process restarts.

```
Client A acquires  →  Token: 7
Client A pauses    (GC pause, network hiccup...)
Client B acquires  →  Token: 8, commits, resource updated to gen 8
Client A resumes   →  tries Commit with Token: 7  →  ErrConflict ✓
```

This is the approach Martin Kleppmann recommends in [Designing Data-Intensive Applications (ch.8)](https://dataintensive.net/) as the correct solution to distributed locking.

### Idempotency keys

Setting `IdempotencyKey` makes `Acquire` safe to retry — the same key always returns the same `Lease`, preventing double-acquisition on network failures.

```
1st call: Acquire(key="order-789")  →  Lease{ ID: "seats-7", Token: 7 }
2nd call: Acquire(key="order-789")  →  Lease{ ID: "seats-7", Token: 7 }  // same lease, no double-hold
```

### TTL and Reap

Leases expire automatically. Run `Reap` periodically to reclaim capacity from dead clients.

```
Acquire(TTL=5min)  →  Expires: 12:00:05
                       ...client crashes...
Reap(now=12:00:06) →  1 lease reclaimed, units restored
```

---

## Backends

| Backend | Package | When to use | Status |
|---|---|---|---|
| In-memory | `store/memory` | Tests, single-process dev | ✅ Available |
| PostgreSQL | `store/postgres` | Production, single primary (streaming replicas OK) | 🚧 In progress |
| Raft | `store/raft` | Multi-node HA, no single point of failure | 📋 Planned |

**On multi-node setups**: The `postgres` backend relies on a single PostgreSQL primary. Streaming replicas are fine for read HA — the primary is the sole source of truth for writes. For a truly leaderless multi-node setup with consensus guarantees, use the `raft` backend.

All backends pass the same **conformance suite**. Implementing a new backend? One call covers all invariants:

```go
func TestMyStore(t *testing.T) {
    conformance.Run(t, mystore.New())
}
```

---

## Usage

### Mode 1 — Embedded library

```go
import (
    "github.com/RIKU-SEINO/holdfast"
    "github.com/RIKU-SEINO/holdfast/store/postgres"
)

store, _ := postgres.New(connString)

store.Register(ctx, holdfast.RegisterRequest{
    Resource: "seats:row-A",
    Capacity: 10,
}, time.Now())

lease, err := store.Acquire(ctx, holdfast.AcquireRequest{
    Resource:       "seats:row-A",
    Units:          2,
    TTL:            5 * time.Minute,
    IdempotencyKey: "order-789",
}, time.Now())
if errors.Is(err, holdfast.ErrExhausted) {
    // no units available
}

store.Commit(ctx, holdfast.CommitRequest{LeaseID: lease.ID, Token: lease.Token})
// or
store.Release(ctx, holdfast.ReleaseRequest{LeaseID: lease.ID, Token: lease.Token})
```

### Mode 2 — gRPC service *(coming soon)*

Run holdfast as a standalone service. Multiple applications — in any language — share the same lease space via generated SDKs.

```
[Go service]      ─┐
[Node.js service]  ─┼── gRPC ──▶ holdfastd ──▶ Store (Postgres / Raft)
[Python service]   ─┘
```

```bash
holdfastd --backend=postgres --dsn="postgres://..."
```

---

## Error reference

| Error | When |
|---|---|
| `ErrExhausted` | `Acquire` was called but available units are insufficient |
| `ErrConflict` | `Commit` or `Release` was called with a stale or unknown token |
| `ErrUnknownResource` | `Acquire` was called on a resource that has not been `Register`ed |

---

## Thread safety

All `Store` implementations are **goroutine-safe**. A single store instance can be shared across goroutines without external synchronization.

---

## Context cancellation

All methods accept a `context.Context`. If the context is cancelled or times out before the operation completes, the method returns immediately with the context's error. No partial state is left behind — an `Acquire` that is cancelled is treated as if it never happened.

---

## Compared to Redlock

[Redlock](https://redis.io/docs/manual/patterns/distributed-locks/) is a distributed locking algorithm on top of Redis. holdfast takes a different approach:

| | Redlock | holdfast |
|---|---|---|
| **Safety model** | Timing-based (assumes bounded clock drift) | Fencing token-based (safe under clock skew and process pauses) |
| **Fencing tokens** | ✗ Not provided | ✓ Every lease carries a monotonically increasing token |
| **Units** | 1 (mutex only) | N (semaphore) |
| **Idempotency** | ✗ | ✓ Via `IdempotencyKey` |
| **TTL reclaim** | Via Redis key expiry | Via explicit `Reap` |
| **Backend** | Redis only | Pluggable (memory / Postgres / Raft) |
| **HA story** | 5 independent Redis nodes (timing assumptions) | Raft consensus (linearizable) |

Kleppmann's [analysis of Redlock](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html) shows that without fencing tokens, a paused client can corrupt shared state even with a "valid" lock. holdfast is built around fencing tokens as a first-class primitive.

---

## Guarantees

Enforced by the conformance suite across all backends:

| Guarantee | Error |
|---|---|
| Total acquired units never exceed registered capacity | `ErrExhausted` |
| Stale or unknown tokens are always rejected | `ErrConflict` |
| Same `IdempotencyKey` always returns the same `Lease` | — |
| Expired leases are reclaimed by `Reap`; capacity is restored | — |
| Safe under concurrent access (`go test -race` clean) | — |

---

## Install

```bash
go get github.com/RIKU-SEINO/holdfast
```

## Testing

```bash
docker compose up -d
docker compose exec dev go test ./...
docker compose exec dev go test -race ./...
```

---

## Contributing

Contributions are welcome. Please read [ROADMAP.md](docs/ROADMAP.md) for the project direction before opening a PR.

When adding a new backend, run the conformance suite against it:

```go
func TestMyStore(t *testing.T) {
    conformance.Run(t, mystore.New())
}
```

---

## License

MIT
