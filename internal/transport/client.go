package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	peerAddr   string
}

func NewClient(peerAddr string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		peerAddr:   peerAddr,
	}
}

func (c *Client) PutChunk(ctx context.Context, chunkID string, data []byte) error {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending chunk %s to %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("peer %s rejected chunk %s: status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return nil
}

func (c *Client) GetChunk(ctx context.Context, chunkID string) ([]byte, error) {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching chunk %s from %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer %s: chunk %s returned status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) DeleteChunk(ctx context.Context, chunkID string) error {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("building delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting chunk %s from %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("peer %s: delete chunk %s returned status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return nil
}

