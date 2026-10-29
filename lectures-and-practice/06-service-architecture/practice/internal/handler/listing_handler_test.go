package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/handler"
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

type fakeSellers struct{}

func (fakeSellers) Get(_ context.Context, id int64) (domain.Seller, error) {
	if id == 2 {
		return domain.NewSeller(2, domain.TierTrusted), nil
	}
	if id == 1 {
		return domain.NewSeller(1, domain.TierRegular), nil
	}
	return domain.Seller{}, domain.ErrSellerNotFound
}

type fakeModerator struct {
	approve bool
	err     error
}

func (f *fakeModerator) Check(_ context.Context, _ int64) (bool, error) {
	return f.approve, f.err
}

type fakeAntifraud struct {
	blocked bool
	err     error
}

func (f *fakeAntifraud) Check(_ context.Context, _ int64, _ float64) (bool, error) {
	return f.blocked, f.err
}

func newMux(mod *fakeModerator, af *fakeAntifraud) http.Handler {
	svc := service.NewListingService(&fakeRepo{}, fakeSellers{}, mod, af)
	h := handler.NewListingHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /listings", h.Create)
	mux.HandleFunc("GET /listings/{id}", h.Get)
	mux.HandleFunc("PATCH /listings/{id}/price", h.UpdatePrice)
	return mux
}

func TestCreate_201Approved(t *testing.T) {
	mux := newMux(&fakeModerator{approve: true}, &fakeAntifraud{})
	req := httptest.NewRequest(http.MethodPost, "/listings", strings.NewReader(
		`{"title":"lamp","price":500,"photo_url":"http://x/1.jpg","seller_id":1}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "approved" {
		t.Errorf("want status approved, got %v", body["status"])
	}
}

func TestCreate_400Price(t *testing.T) {
	mux := newMux(&fakeModerator{approve: true}, &fakeAntifraud{})
	req := httptest.NewRequest(http.MethodPost, "/listings", strings.NewReader(
		`{"title":"lamp","price":0,"photo_url":"http://x/1.jpg","seller_id":1}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "price must be positive") {
		t.Errorf("unexpected body %q", rec.Body.String())
	}
}

func TestGet_404EmptyBody(t *testing.T) {
	mux := newMux(&fakeModerator{}, &fakeAntifraud{})
	req := httptest.NewRequest(http.MethodGet, "/listings/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if len(body) != 0 {
		t.Errorf("404 must have empty body, got %q", body)
	}
}

func TestUpdatePrice_409Rejected(t *testing.T) {
	mux := newMux(&fakeModerator{approve: false}, &fakeAntifraud{})
	create := httptest.NewRequest(http.MethodPost, "/listings", strings.NewReader(
		`{"title":"lamp","price":500,"photo_url":"http://x/1.jpg","seller_id":1}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create: %d %s", createRec.Code, createRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPatch, "/listings/1/price", strings.NewReader(`{"price":700}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cannot update rejected listing") {
		t.Errorf("unexpected body %q", rec.Body.String())
	}
}
