package service

import (
	"context"
	"errors"
	"time"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

const moderationTimeout = 2 * time.Second

type ListingService struct {
	repo      ListingRepository
	sellers   SellerRepository
	moderator Moderator
	antifraud Antifraud
}

func NewListingService(
	repo ListingRepository,
	sellers SellerRepository,
	moderator Moderator,
	antifraud Antifraud,
) *ListingService {
	return &ListingService{
		repo:      repo,
		sellers:   sellers,
		moderator: moderator,
		antifraud: antifraud,
	}
}

type PublishInput struct {
	Title    string
	Price    float64
	PhotoURL string
	SellerID int64
}

func (s *ListingService) Publish(ctx context.Context, in PublishInput) (*domain.Listing, error) {
	listing, err := domain.NewListing(in.Title, in.Price, in.PhotoURL, in.SellerID)
	if err != nil {
		return nil, err
	}

	seller, err := s.resolveSeller(ctx, in.SellerID)
	if err != nil {
		return nil, err
	}
	if err := listing.CanPublishFor(seller); err != nil {
		return nil, err
	}

	blocked, err := s.antifraud.Check(ctx, in.SellerID, in.Price)
	if err != nil {
		blocked = false
	}

	id, err := s.repo.Create(ctx, listing)
	if err != nil {
		return nil, err
	}
	listing.ID = id

	if blocked {
		if err := listing.Reject(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(ctx, listing); err != nil {
			return nil, err
		}
		return listing, nil
	}

	if seller.SkipsModeration() {
		if err := listing.Approve(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(ctx, listing); err != nil {
			return nil, err
		}
		return listing, nil
	}

	modCtx, cancel := context.WithTimeout(ctx, moderationTimeout)
	defer cancel()

	approved, err := s.moderator.Check(modCtx, listing.ID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			if err := listing.MarkReview(); err != nil {
				return nil, err
			}
			if err := s.repo.Save(ctx, listing); err != nil {
				return nil, err
			}
		}
		return listing, nil
	}

	if approved {
		if err := listing.Approve(); err != nil {
			return nil, err
		}
	} else {
		if err := listing.Reject(); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Save(ctx, listing); err != nil {
		return nil, err
	}
	return listing, nil
}

func (s *ListingService) Get(ctx context.Context, id int64) (*domain.Listing, error) {
	listing, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return listing, nil
}

func (s *ListingService) ChangePrice(ctx context.Context, id int64, price float64) (*domain.Listing, error) {
	listing, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	seller, err := s.resolveSeller(ctx, listing.SellerID)
	if err != nil {
		return nil, err
	}

	if err := listing.ChangePrice(price, seller); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, listing); err != nil {
		return nil, err
	}
	return listing, nil
}

func (s *ListingService) resolveSeller(ctx context.Context, id int64) (domain.Seller, error) {
	seller, err := s.sellers.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrSellerNotFound) {
			return domain.RegularSeller(id), nil
		}
		return domain.Seller{}, err
	}
	return seller, nil
}
