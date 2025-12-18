package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

type Request struct {
	Message string `json:"message"`
	User    string `json:"user"`
}

type Response struct {
	Response string `json:"response"`
}

func New(endpoint, apiKey string, timeout time.Duration) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Send(ctx context.Context, message, user string) (string, error) {
	reqBody := Request{
		Message: message,
		User:    user,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	slog.Debug("sending api request",
		"endpoint", c.endpoint,
		"user", user,
		"body_size", len(body),
	)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("api request failed",
			"endpoint", c.endpoint,
			"user", user,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	slog.Debug("api response received",
		"endpoint", c.endpoint,
		"user", user,
		"status_code", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body",
			"endpoint", c.endpoint,
			"user", user,
			"error", err,
		)
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("api returned non-ok status",
			"endpoint", c.endpoint,
			"user", user,
			"status_code", resp.StatusCode,
			"response_body", string(bodyBytes),
		)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result Response
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		slog.Error("failed to decode response",
			"endpoint", c.endpoint,
			"user", user,
			"response_body", string(bodyBytes),
			"error", err,
		)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Response == "" {
		slog.Warn("api returned empty response",
			"endpoint", c.endpoint,
			"user", user,
			"response_body", string(bodyBytes),
		)
	}

	return result.Response, nil
}
