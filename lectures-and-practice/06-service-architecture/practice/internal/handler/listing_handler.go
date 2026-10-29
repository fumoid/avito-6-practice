package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/service"
)

type ListingHandler struct {
	service *service.ListingService
}

func NewListingHandler(s *service.ListingService) *ListingHandler {
	return &ListingHandler{service: s}
}

type createRequest struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	PhotoURL string  `json:"photo_url"`
	SellerID int64   `json:"seller_id"`
}

type updatePriceRequest struct {
	Price float64 `json:"price"`
}

func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.Publish(r.Context(), service.PublishInput{
		Title:    req.Title,
		Price:    req.Price,
		PhotoURL: req.PhotoURL,
		SellerID: req.SellerID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": listing.ID, "status": listing.Status})
}

func (h *ListingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

func (h *ListingHandler) UpdatePrice(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req updatePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, err := h.service.ChangePrice(r.Context(), id, req.Price)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrPhotoMissing),
		errors.Is(err, domain.ErrSuspiciousPrice):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, domain.ErrCannotUpdateRejected):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
