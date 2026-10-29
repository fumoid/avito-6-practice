package domain

import (
	"errors"
	"strings"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusReview   Status = "review"
)

var (
	ErrInvalidPrice         = errors.New("price must be positive")
	ErrPhotoMissing         = errors.New("photo is required")
	ErrSuspiciousPrice      = errors.New("price looks suspicious")
	ErrNotFound             = errors.New("listing not found")
	ErrCannotUpdateRejected = errors.New("cannot update rejected listing")
)

type Listing struct {
	ID        int64
	Title     string
	Price     float64
	PhotoURL  string
	SellerID  int64
	Status    Status
	CreatedAt time.Time
}

func NewListing(title string, price float64, photoURL string, sellerID int64) (*Listing, error) {
	if price <= 0 {
		return nil, ErrInvalidPrice
	}
	if strings.TrimSpace(photoURL) == "" {
		return nil, ErrPhotoMissing
	}
	return &Listing{
		Title:     title,
		Price:     price,
		PhotoURL:  photoURL,
		SellerID:  sellerID,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}, nil
}

func (l *Listing) CanPublishFor(seller Seller) error {
	if l.Price > seller.SuspiciousPriceLimit() {
		return ErrSuspiciousPrice
	}
	return nil
}

func (l *Listing) Approve() error {
	if l.Status != StatusPending {
		return errors.New("only pending listing can be approved")
	}
	l.Status = StatusApproved
	return nil
}

func (l *Listing) Reject() error {
	if l.Status != StatusPending {
		return errors.New("only pending listing can be rejected")
	}
	l.Status = StatusRejected
	return nil
}

func (l *Listing) MarkReview() error {
	if l.Status != StatusPending {
		return errors.New("only pending listing can move to review")
	}
	l.Status = StatusReview
	return nil
}

func (l *Listing) ChangePrice(price float64, seller Seller) error {
	if price <= 0 {
		return ErrInvalidPrice
	}
	if l.Status == StatusRejected {
		return ErrCannotUpdateRejected
	}
	l.Price = price
	if l.Status == StatusApproved && price > seller.SuspiciousPriceLimit() {
		l.Status = StatusPending
	}
	return nil
}
