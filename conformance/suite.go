package conformance

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RIKU-SEINO/holdfast"
)

// Run はどのバックエンドにも通す適合テスト群。
// 新しいバックエンドを作ったら conformance.Run(t, store) を呼ぶだけでよい。
func Run(t *testing.T, store holdfast.Store) {
	t.Helper()
	t.Run("枠を超えて確保できない", func(t *testing.T) { testExhausted(t, store) })
	t.Run("古いtokenは弾かれる", func(t *testing.T) { testStaleToken(t, store) })
	t.Run("TTLが切れると枠が戻る", func(t *testing.T) { testReap(t, store) })
	t.Run("複数のgoroutineから同時にAcquireを呼び出す", func(t *testing.T) { testConcurrentAcquire(t, store) })
}

// testExhausted: 枠を超えて確保することはできない
func testExhausted(t *testing.T, store holdfast.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	regReq := holdfast.RegisterRequest{
		Resource: "test:exhausted",
		Capacity: 1,
	}
	err := store.Register(ctx, regReq, now)
	if err != nil {
		t.Fatalf("Register が失敗した: %v", err)
	}

	acqReq := holdfast.AcquireRequest{
		Resource: "test:exhausted",
		Units:    1,
		TTL:      time.Minute,
	}

	_, err = store.Acquire(ctx, acqReq, now)
	if err != nil {
		t.Fatalf("1回目の Acquire が失敗した: %v", err)
	}

	_, err = store.Acquire(ctx, acqReq, now)
	if err != holdfast.ErrExhausted {
		t.Fatalf("2回目の Acquire は ErrExhausted を期待したが got: %v", err)
	}
}

// testStaleToken: 古いtokenは弾かれる
func testStaleToken(t *testing.T, store holdfast.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	regReq := holdfast.RegisterRequest{
		Resource: "test:stale",
		Capacity: 1,
	}
	err := store.Register(ctx, regReq, now)
	if err != nil {
		t.Fatalf("Register が失敗した: %v", err)
	}

	acqReq := holdfast.AcquireRequest{
		Resource: "test:stale",
		Units:    1,
		TTL:      time.Minute,
	}
	lease, err := store.Acquire(ctx, acqReq, now)
	if err != nil {
		t.Fatalf("Acquire が失敗した: %v", err)
	}

	comReq := holdfast.CommitRequest{
		LeaseID: lease.ID,
		Token:   lease.Token - 1,
	}
	_, err = store.Commit(ctx, comReq)
	if err != holdfast.ErrConflict {
		t.Fatalf("Commit は ErrConflict を期待したが got: %v", err)
	}
}

// testReap: TTLが切れると枠が戻る
func testReap(t *testing.T, store holdfast.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	regReq := holdfast.RegisterRequest{
		Resource: "test:reap",
		Capacity: 1,
	}
	err := store.Register(ctx, regReq, now)
	if err != nil {
		t.Fatalf("Register が失敗した: %v", err)
	}

	acqReq := holdfast.AcquireRequest{
		Resource: "test:reap",
		Units:    1,
		TTL:      time.Minute,
	}

	lease, err := store.Acquire(ctx, acqReq, now)
	if err != nil {
		t.Fatalf("Acquire が失敗した: %v", err)
	}

	leaseExpires := lease.Expires
	_, err = store.Reap(ctx, leaseExpires.Add(time.Nanosecond))
	if err != nil {
		t.Fatalf("Reap が失敗した: %v", err)
	}

	_, err = store.Acquire(ctx, acqReq, leaseExpires.Add(time.Nanosecond))
	if err != nil {
		t.Fatalf("Reap 後の Acquire が失敗した: %v", err)
	}
}

// testConcurrentAcquire: 複数のgoroutineから同時にAcquireを呼び出す
func testConcurrentAcquire(t *testing.T, store holdfast.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	const capacity = 10
	const goroutines = 100

	regReq := holdfast.RegisterRequest{
		Resource: "test:concurrent",
		Capacity: capacity,
	}
	err := store.Register(ctx, regReq, now)
	if err != nil {
		t.Fatalf("Register が失敗した: %v", err)
	}

	acqReq := holdfast.AcquireRequest{
		Resource: "test:concurrent",
		Units:    1,
		TTL:      time.Minute,
	}

	var success atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_, err := store.Acquire(ctx, acqReq, now)
			if err == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()

	var loaded = success.Load()

	if loaded > capacity {
		t.Fatalf("Acquire 成功数が capacity を超えた: got %d, want <= %d", loaded, capacity)
	}
}
