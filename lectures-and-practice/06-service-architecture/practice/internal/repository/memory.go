package repository

import (
	"context"
	"sync"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	items  map[int64]listingRecord
	nextID int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items: make(map[int64]listingRecord),
	}
}

func (r *MemoryRepository) Create(
	_ context.Context,
	listing *domain.Listing,
) (*domain.Listing, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := recordFromDomain(listing)
	record.ID = r.nextID + 1

	created, err := record.toDomain()
	if err != nil {
		return nil, err
	}

	r.nextID = record.ID
	r.items[record.ID] = record

	return created, nil
}

func (r *MemoryRepository) Get(
	_ context.Context,
	id int64,
) (*domain.Listing, error) {
	r.mu.RLock()
	record, ok := r.items[id]
	r.mu.RUnlock()

	if !ok {
		return nil, domain.ErrNotFound
	}

	return record.toDomain()
}

func (r *MemoryRepository) Save(
	_ context.Context,
	listing *domain.Listing,
) error {
	record := recordFromDomain(listing)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[record.ID]; !ok {
		return domain.ErrNotFound
	}

	r.items[record.ID] = record
	return nil
}
