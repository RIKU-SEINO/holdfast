package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/RIKU-SEINO/holdfast"
)

type leaseRecord struct {
	resource	string
	units			int
	token			uint64
	expires		time.Time
}

func (l leaseRecord) isExpired(now time.Time) bool {
	return now.After(l.expires)
}

type resourceState struct {
	capacity	int
	used			int
}

func (r resourceState) isExhausted(units int) bool {
	return r.used+units > r.capacity
}

type MemoryStore struct {
	leases		map[string]leaseRecord
	resources	map[string]resourceState
	nextToken	uint64
}

func New() *MemoryStore {
	return &MemoryStore{
		leases:    make(map[string]leaseRecord),
		resources: make(map[string]resourceState),
	}
}

func (s *MemoryStore) Register(ctx context.Context, req holdfast.RegisterRequest, now time.Time) error {
	if _, exists := s.resources[req.Resource]; exists {
		return nil
	}

	s.resources[req.Resource] = resourceState{
		capacity: req.Capacity,
		used:     0,
	}
	return nil
}

func (s *MemoryStore) Acquire(ctx context.Context, req holdfast.AcquireRequest, now time.Time) (holdfast.Lease, error) {
	if _, exists := s.resources[req.Resource]; !exists {
		return holdfast.Lease{}, holdfast.ErrUnknownResource
	}

	state := s.resources[req.Resource]
	units := req.Units
	if state.isExhausted(units) {
		return holdfast.Lease{}, holdfast.ErrExhausted
	}

	state.used += units
	s.resources[req.Resource] = state

	token := s.nextToken
	leaseId := fmt.Sprintf("%s-%d", req.Resource, token)
	s.nextToken++

	expires := now.Add(req.TTL)
	s.leases[leaseId] = leaseRecord{
		resource: req.Resource,
		units:    units,
		token:    token,
		expires:  expires,
	}

	return holdfast.Lease{
		ID:      leaseId,
		Token:   token,
		Expires: expires,
	}, nil
}

func (s *MemoryStore) validateLease(leaseID string, token uint64) (leaseRecord, error) {
	lease, exists := s.leases[leaseID]
	if !exists || lease.token != token {
		return leaseRecord{}, holdfast.ErrConflict
	}
	return lease, nil
}

func (s *MemoryStore) Commit(ctx context.Context, req holdfast.CommitRequest) (holdfast.Receipt, error) {
	if _, err := s.validateLease(req.LeaseID, req.Token); err != nil {
		return holdfast.Receipt{}, err
	}
	return holdfast.Receipt{LeaseID: req.LeaseID}, nil
}

func (s *MemoryStore) Release(ctx context.Context, req holdfast.ReleaseRequest) error {
	lease, err := s.validateLease(req.LeaseID, req.Token)
	if err != nil {
		return err
	}

	state := s.resources[lease.resource]
	state.used -= lease.units
	s.resources[lease.resource] = state

	delete(s.leases, req.LeaseID)

	return nil
}

func (s *MemoryStore) Reap(ctx context.Context, now time.Time) (int, error) {
	count := 0
	for leaseId, leaseRecord := range s.leases {
		if !leaseRecord.isExpired(now) {
			continue
		}
		resource := leaseRecord.resource
		units := leaseRecord.units
		state := s.resources[resource]
		state.used -= units
		s.resources[resource] = state

		delete(s.leases, leaseId)

		count++
	}

	return count, nil
}