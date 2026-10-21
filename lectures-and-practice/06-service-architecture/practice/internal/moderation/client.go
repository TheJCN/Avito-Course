package moderation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	url string
}

func NewClient(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Check(ctx context.Context, listingID int64) (bool, error) {
	body := strings.NewReader(fmt.Sprintf(`{"listing_id":%d}`, listingID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
