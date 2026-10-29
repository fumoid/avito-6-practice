package main

import (
	"log"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/handler"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/repository"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/service"
)

func main() {
	listings := repository.NewMemoryListingRepository()
	sellers := repository.NewMemorySellerRepository()
	moderator := repository.NewHTTPModerator("http://moderation.internal/check")
	antifraud := repository.NewHTTPAntifraud("http://antifraud.internal/check")

	svc := service.NewListingService(listings, sellers, moderator, antifraud)
	h := handler.NewListingHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /listings", h.Create)
	mux.HandleFunc("GET /listings/{id}", h.Get)
	mux.HandleFunc("PATCH /listings/{id}/price", h.UpdatePrice)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
