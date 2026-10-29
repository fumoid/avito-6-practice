package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type HTTPAntifraud struct {
	url    string
	client *http.Client
}

func NewHTTPAntifraud(url string) *HTTPAntifraud {
	return &HTTPAntifraud{url: url, client: http.DefaultClient}
}

func (a *HTTPAntifraud) Check(ctx context.Context, sellerID int64, price float64) (bool, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.url,
		strings.NewReader(fmt.Sprintf(`{"seller_id":%d,"price":%f}`, sellerID, price)),
	)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusForbidden, nil
}
