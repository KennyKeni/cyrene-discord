package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type StreamClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewStreamClient(endpoint, apiKey string, timeout time.Duration) *StreamClient {
	return &StreamClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *StreamClient) SendStream(ctx context.Context, message, user string, onChunk func(chunk string)) error {
	reqBody := Request{
		Message: message,
		User:    user,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	slog.Debug("sending stream request",
		"endpoint", c.endpoint,
		"user", user,
		"body_size", len(body),
	)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("stream request failed",
			"endpoint", c.endpoint,
			"user", user,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("stream returned non-ok status",
			"endpoint", c.endpoint,
			"user", user,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: error") {
			if scanner.Scan() {
				errorLine := scanner.Text()
				errorMsg := strings.TrimPrefix(errorLine, "data: ")
				slog.Error("stream error event",
					"endpoint", c.endpoint,
					"user", user,
					"error", errorMsg,
				)
				return fmt.Errorf("stream error: %s", errorMsg)
			}
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			slog.Debug("stream completed",
				"endpoint", c.endpoint,
				"user", user,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			break
		}

		onChunk(data)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("stream scanner error",
			"endpoint", c.endpoint,
			"user", user,
			"error", err,
		)
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}
