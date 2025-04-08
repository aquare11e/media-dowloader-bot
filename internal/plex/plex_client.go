package plex

import (
	"context"
	"fmt"
	"net/http"

	common "github.com/aquare11e/media-downloader-bot/common/protogen/common"
)

// Client represents a Plex client
type Client struct {
	baseURL            string
	token              string
	pbTypeToCategoryId map[common.RequestType]string
	httpClient         *http.Client
}

// NewClient creates a new Plex client
func NewClient(baseURL string, token string, pbTypeToCategoryId map[common.RequestType]string) *Client {
	return &Client{
		baseURL:            baseURL,
		token:              token,
		pbTypeToCategoryId: pbTypeToCategoryId,
		httpClient:         &http.Client{},
	}
}

// ScanLibrary scans a single Plex library
func (c *Client) ScanLibrary(ctx context.Context, pbType common.RequestType) error {
	categoryId, ok := c.pbTypeToCategoryId[pbType]
	if !ok {
		return fmt.Errorf("category id not found for pb type: %d", pbType)
	}

	scanURL := fmt.Sprintf("%s/library/sections/%s/refresh?X-Plex-Token=%s", c.baseURL, categoryId, c.token)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", scanURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
