package repository

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type MemoryListingRepository struct {
	items  map[int64]*domain.Listing
	nextID int64
}

func NewMemoryListingRepository() *MemoryListingRepository {
	return &MemoryListingRepository{items: map[int64]*domain.Listing{}}
}

func (r *MemoryListingRepository) Create(_ context.Context, l *domain.Listing) (int64, error) {
	r.nextID++
	id := r.nextID
	stored := *l
	stored.ID = id
	r.items[id] = &stored
	return id, nil
}

func (r *MemoryListingRepository) Get(_ context.Context, id int64) (*domain.Listing, error) {
	stored, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *stored
	return &clone, nil
}

func (r *MemoryListingRepository) Save(_ context.Context, l *domain.Listing) error {
	if _, ok := r.items[l.ID]; !ok {
		return domain.ErrNotFound
	}
	stored := *l
	r.items[l.ID] = &stored
	return nil
}
