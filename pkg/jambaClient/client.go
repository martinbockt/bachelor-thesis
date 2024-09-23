package jambaClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

const apiVersion = "/v1"

type client struct {
	apiKey  string
	baseURL string
	logger  *zap.Logger
	client  *http.Client
}

// NewClient initializes a new TogetherAI client.
func newClient(logger *zap.Logger, apiKey string) *client {
	return &client{
		apiKey:  apiKey,
		baseURL: "https://api.ai21.com/studio",
		logger:  logger,
		client:  &http.Client{
			// Timeout: 120 * time.Second,
			// Transport: &http.Transport{
			// 	MaxIdleConns:    10,
			// 	IdleConnTimeout: 120 * time.Second,
			// },
		},
	}
}

// handleResponse handles the HTTP response and decodes the JSON into the response interface.
func (c *client) handleResponse(resp *http.Response, res interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("Jamba request failed",
			zap.Int("status", resp.StatusCode),
			zap.String("url", resp.Request.URL.String()),
			zap.String("method", resp.Request.Method),
			zap.String("response", string(body)),
		)

		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// post sends a POST request with the specified body and decodes the response.
func (c *client) post(ctx context.Context, url string, body, res interface{}) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+apiVersion+url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	// fmt.Println(resp)
	// respBody, readErr := io.ReadAll(resp.Body)
	// if readErr != nil {
	// 	fmt.Println("Error reading response body:", readErr)
	// 	return nil
	// }

	// // Print the response body
	// fmt.Println(string(respBody))
	return c.handleResponse(resp, res)
}
