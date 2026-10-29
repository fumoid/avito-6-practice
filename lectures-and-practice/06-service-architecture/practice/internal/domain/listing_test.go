package domain

import "testing"

func TestNewListing_InvalidPrice(t *testing.T) {
	_, err := NewListing("lamp", 0, "http://x/1.jpg", 1)
	if err != ErrInvalidPrice {
		t.Fatalf("want ErrInvalidPrice, got %v", err)
	}
}

func TestNewListing_PhotoMissing(t *testing.T) {
	_, err := NewListing("lamp", 500, "   ", 1)
	if err != ErrPhotoMissing {
		t.Fatalf("want ErrPhotoMissing, got %v", err)
	}
}

func TestCanPublishFor_SuspiciousPriceDependsOnTier(t *testing.T) {
	listing, err := NewListing("car", 20_000_000, "http://x/2.jpg", 1)
	if err != nil {
		t.Fatal(err)
	}

	if err := listing.CanPublishFor(RegularSeller(1)); err != ErrSuspiciousPrice {
		t.Fatalf("regular seller: want ErrSuspiciousPrice, got %v", err)
	}
	if err := listing.CanPublishFor(NewSeller(2, TierTrusted)); err != nil {
		t.Fatalf("trusted seller: unexpected error %v", err)
	}
}

func TestChangePrice_RejectedForbidden(t *testing.T) {
	listing, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	if err := listing.Reject(); err != nil {
		t.Fatal(err)
	}
	err := listing.ChangePrice(700, RegularSeller(1))
	if err != ErrCannotUpdateRejected {
		t.Fatalf("want ErrCannotUpdateRejected, got %v", err)
	}
}

func TestChangePrice_ApprovedRollsBackOnSuspiciousPrice(t *testing.T) {
	listing, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	if err := listing.Approve(); err != nil {
		t.Fatal(err)
	}

	if err := listing.ChangePrice(20_000_000, RegularSeller(1)); err != nil {
		t.Fatal(err)
	}
	if listing.Status != StatusPending {
		t.Fatalf("want pending after suspicious price, got %s", listing.Status)
	}
	if listing.Price != 20_000_000 {
		t.Fatalf("want price 20000000, got %v", listing.Price)
	}
}

func TestChangePrice_ApprovedStaysIfWithinLimit(t *testing.T) {
	listing, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	_ = listing.Approve()

	if err := listing.ChangePrice(700, RegularSeller(1)); err != nil {
		t.Fatal(err)
	}
	if listing.Status != StatusApproved {
		t.Fatalf("want approved, got %s", listing.Status)
	}
}

func TestChangePrice_PendingAndReviewKeepStatus(t *testing.T) {
	pending, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	if err := pending.ChangePrice(800, RegularSeller(1)); err != nil {
		t.Fatal(err)
	}
	if pending.Status != StatusPending {
		t.Fatalf("pending must stay pending, got %s", pending.Status)
	}

	review, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	if err := review.MarkReview(); err != nil {
		t.Fatal(err)
	}
	if err := review.ChangePrice(800, RegularSeller(1)); err != nil {
		t.Fatal(err)
	}
	if review.Status != StatusReview {
		t.Fatalf("review must stay review, got %s", review.Status)
	}
}

func TestApprove_OnlyFromPending(t *testing.T) {
	listing, _ := NewListing("lamp", 500, "http://x/1.jpg", 1)
	_ = listing.Reject()
	if err := listing.Approve(); err == nil {
		t.Fatal("expected error when approving rejected listing")
	}
}
