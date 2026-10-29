package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type HTTPModerator struct {
	url    string
	client *http.Client
}

func NewHTTPModerator(url string) *HTTPModerator {
	return &HTTPModerator{url: url, client: http.DefaultClient}
}

func (m *HTTPModerator) Check(ctx context.Context, listingID int64) (bool, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		m.url,
		strings.NewReader(fmt.Sprintf(`{"listing_id":%d}`, listingID)),
	)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
