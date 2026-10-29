package service

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type ListingRepository interface {
	Create(ctx context.Context, l *domain.Listing) (int64, error)
	Get(ctx context.Context, id int64) (*domain.Listing, error)
	Save(ctx context.Context, l *domain.Listing) error
}

type SellerRepository interface {
	Get(ctx context.Context, id int64) (domain.Seller, error)
}

type Moderator interface {
	Check(ctx context.Context, listingID int64) (approved bool, err error)
}

type Antifraud interface {
	Check(ctx context.Context, sellerID int64, price float64) (blocked bool, err error)
}
