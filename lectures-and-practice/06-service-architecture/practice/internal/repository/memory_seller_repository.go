package repository

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type MemorySellerRepository struct {
	items map[int64]domain.Seller
}

func NewMemorySellerRepository() *MemorySellerRepository {
	return &MemorySellerRepository{
		items: map[int64]domain.Seller{
			1: domain.NewSeller(1, domain.TierRegular),
			2: domain.NewSeller(2, domain.TierTrusted),
		},
	}
}

func (r *MemorySellerRepository) Get(_ context.Context, id int64) (domain.Seller, error) {
	seller, ok := r.items[id]
	if !ok {
		return domain.Seller{}, domain.ErrSellerNotFound
	}
	return seller, nil
}
