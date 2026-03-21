package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type ControlPlaneClient struct {
	baseURL string
	client  *http.Client
}

func NewControlPlaneClient(baseURL string) *ControlPlaneClient {
	return &ControlPlaneClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ControlPlaneClient) Snapshot(ctx context.Context) (domain.PlatformData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/internal/platform-data", nil)
	if err != nil {
		return domain.PlatformData{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return domain.PlatformData{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.PlatformData{}, fmt.Errorf("snapshot request failed: %s", resp.Status)
	}
	var data domain.PlatformData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (c *ControlPlaneClient) AppendUsageLog(ctx context.Context, usageLog domain.UsageLog) error {
	body, err := json.Marshal(usageLog)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/internal/usage-logs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("usage log append failed: %s", resp.Status)
	}
	return nil
}
