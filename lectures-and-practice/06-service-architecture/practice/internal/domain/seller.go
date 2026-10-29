package domain

import "errors"

type Tier string

const (
	TierRegular Tier = "regular"
	TierTrusted Tier = "trusted"

	regularPriceLimit = 10_000_000.0
	trustedPriceLimit = 50_000_000.0
)

var ErrSellerNotFound = errors.New("seller not found")

type Seller struct {
	ID   int64
	Tier Tier
}

func NewSeller(id int64, tier Tier) Seller {
	if tier != TierTrusted {
		tier = TierRegular
	}
	return Seller{ID: id, Tier: tier}
}

func RegularSeller(id int64) Seller {
	return NewSeller(id, TierRegular)
}

func (s Seller) SuspiciousPriceLimit() float64 {
	if s.Tier == TierTrusted {
		return trustedPriceLimit
	}
	return regularPriceLimit
}

func (s Seller) SkipsModeration() bool {
	return s.Tier == TierTrusted
}
