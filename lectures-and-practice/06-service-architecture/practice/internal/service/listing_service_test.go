package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/service"
)

type fakeRepo struct {
	stored *domain.Listing
	nextID int64
}

func (f *fakeRepo) Create(_ context.Context, l *domain.Listing) (int64, error) {
	f.nextID++
	clone := *l
	clone.ID = f.nextID
	f.stored = &clone
	return clone.ID, nil
}

func (f *fakeRepo) Get(_ context.Context, id int64) (*domain.Listing, error) {
	if f.stored == nil || f.stored.ID != id {
		return nil, domain.ErrNotFound
	}
	clone := *f.stored
	return &clone, nil
}

func (f *fakeRepo) Save(_ context.Context, l *domain.Listing) error {
	if f.stored == nil || f.stored.ID != l.ID {
		return domain.ErrNotFound
	}
	clone := *l
	f.stored = &clone
	return nil
}

type fakeSellers struct {
	byID map[int64]domain.Seller
}

func defaultSellers() *fakeSellers {
	return &fakeSellers{byID: map[int64]domain.Seller{
		1: domain.NewSeller(1, domain.TierRegular),
		2: domain.NewSeller(2, domain.TierTrusted),
	}}
}

func (f *fakeSellers) Get(_ context.Context, id int64) (domain.Seller, error) {
	s, ok := f.byID[id]
	if !ok {
		return domain.Seller{}, domain.ErrSellerNotFound
	}
	return s, nil
}

type fakeModerator struct {
	approve bool
	err     error
	calls   int
}

func (f *fakeModerator) Check(_ context.Context, _ int64) (bool, error) {
	f.calls++
	return f.approve, f.err
}

type fakeAntifraud struct {
	blocked bool
	err     error
	calls   int
}

func (f *fakeAntifraud) Check(_ context.Context, _ int64, _ float64) (bool, error) {
	f.calls++
	return f.blocked, f.err
}

func newService(mod *fakeModerator, af *fakeAntifraud) (*service.ListingService, *fakeRepo) {
	repo := &fakeRepo{}
	return service.NewListingService(repo, defaultSellers(), mod, af), repo
}

func publish(t *testing.T, svc *service.ListingService, in service.PublishInput) *domain.Listing {
	t.Helper()
	listing, err := svc.Publish(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return listing
}

func TestPublish_Approved(t *testing.T) {
	mod := &fakeModerator{approve: true}
	svc, _ := newService(mod, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if listing.Status != domain.StatusApproved {
		t.Errorf("want approved, got %s", listing.Status)
	}
	if mod.calls != 1 {
		t.Errorf("moderator should be called once, got %d", mod.calls)
	}
}

func TestPublish_RejectedByModerator(t *testing.T) {
	svc, _ := newService(&fakeModerator{approve: false}, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if listing.Status != domain.StatusRejected {
		t.Errorf("want rejected, got %s", listing.Status)
	}
}

func TestPublish_InvalidPrice(t *testing.T) {
	svc, _ := newService(&fakeModerator{approve: true}, &fakeAntifraud{})

	_, err := svc.Publish(context.Background(), service.PublishInput{
		Title: "lamp", Price: 0, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if err != domain.ErrInvalidPrice {
		t.Errorf("want ErrInvalidPrice, got %v", err)
	}
}

func TestPublish_SuspiciousPrice_Regular(t *testing.T) {
	svc, _ := newService(&fakeModerator{approve: true}, &fakeAntifraud{})

	_, err := svc.Publish(context.Background(), service.PublishInput{
		Title: "car", Price: 20_000_000, PhotoURL: "http://x/2.jpg", SellerID: 1,
	})
	if err != domain.ErrSuspiciousPrice {
		t.Errorf("want ErrSuspiciousPrice, got %v", err)
	}
}

func TestPublish_HighPrice_TrustedAllowed(t *testing.T) {
	mod := &fakeModerator{approve: false}
	svc, _ := newService(mod, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "car", Price: 20_000_000, PhotoURL: "http://x/2.jpg", SellerID: 2,
	})
	if listing.Status != domain.StatusApproved {
		t.Errorf("trusted seller must skip moderation, got %s", listing.Status)
	}
	if mod.calls != 0 {
		t.Errorf("moderator must not be called for trusted seller, got %d", mod.calls)
	}
}

func TestPublish_UnknownSeller_TreatedAsRegular(t *testing.T) {
	mod := &fakeModerator{approve: true}
	svc, _ := newService(mod, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 99,
	})
	if listing.Status != domain.StatusApproved {
		t.Errorf("want approved after moderation, got %s", listing.Status)
	}
	if mod.calls != 1 {
		t.Errorf("unknown seller is regular and must go through moderation")
	}
}

func TestPublish_AntifraudBlocked_SkipsModeration(t *testing.T) {
	mod := &fakeModerator{approve: true}
	svc, _ := newService(mod, &fakeAntifraud{blocked: true})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 2,
	})
	if listing.Status != domain.StatusRejected {
		t.Errorf("want rejected, got %s", listing.Status)
	}
	if mod.calls != 0 {
		t.Errorf("moderation must not run after antifraud block, got %d calls", mod.calls)
	}
}

func TestPublish_AntifraudDown_FailOpen(t *testing.T) {
	mod := &fakeModerator{approve: true}
	svc, _ := newService(mod, &fakeAntifraud{err: errors.New("antifraud down")})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if listing.Status != domain.StatusApproved {
		t.Errorf("fail-open antifraud should continue to moderation, got %s", listing.Status)
	}
}

func TestPublish_ModerationDown_StaysPending(t *testing.T) {
	svc, _ := newService(&fakeModerator{err: errors.New("network")}, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if listing.Status != domain.StatusPending {
		t.Errorf("want pending when moderation is down, got %s", listing.Status)
	}
}

func TestPublish_ModerationTimeout_Review(t *testing.T) {
	svc, _ := newService(&fakeModerator{err: context.DeadlineExceeded}, &fakeAntifraud{})

	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if listing.Status != domain.StatusReview {
		t.Errorf("want review on moderation timeout, got %s", listing.Status)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := newService(&fakeModerator{}, &fakeAntifraud{})

	_, err := svc.Get(context.Background(), 1)
	if err != domain.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestChangePrice_RejectedConflict(t *testing.T) {
	svc, repo := newService(&fakeModerator{approve: false}, &fakeAntifraud{})
	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})
	if repo.stored.Status != domain.StatusRejected {
		t.Fatal("precondition: listing must be rejected")
	}

	_, err := svc.ChangePrice(context.Background(), listing.ID, 700)
	if err != domain.ErrCannotUpdateRejected {
		t.Errorf("want ErrCannotUpdateRejected, got %v", err)
	}
}

func TestChangePrice_ApprovedSuspicious_GoesPending(t *testing.T) {
	svc, _ := newService(&fakeModerator{approve: true}, &fakeAntifraud{})
	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})

	updated, err := svc.ChangePrice(context.Background(), listing.ID, 20_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusPending {
		t.Errorf("want pending, got %s", updated.Status)
	}
	if updated.Price != 20_000_000 {
		t.Errorf("want new price, got %v", updated.Price)
	}
}

func TestChangePrice_ApprovedWithinLimit_StaysApproved(t *testing.T) {
	svc, _ := newService(&fakeModerator{approve: true}, &fakeAntifraud{})
	listing := publish(t, svc, service.PublishInput{
		Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1,
	})

	updated, err := svc.ChangePrice(context.Background(), listing.ID, 700)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusApproved {
		t.Errorf("want approved, got %s", updated.Status)
	}
}

func TestChangePrice_NotFound(t *testing.T) {
	svc, _ := newService(&fakeModerator{}, &fakeAntifraud{})

	_, err := svc.ChangePrice(context.Background(), 99, 700)
	if err != domain.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
