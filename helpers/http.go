package helpers

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient wraps http.Client with retry logic
type HTTPClient struct {
	client     *http.Client
	maxRetries int
}

// NewHTTPClient creates an HTTP client with a timeout and retry count
func NewHTTPClient(timeout time.Duration, maxRetries int) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// Get performs a GET request with retries
func (c *HTTPClient) Get(url string, headers map[string]string) ([]byte, int, error) {
	var lastErr error

	for i := 0; i <= c.maxRetries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, 0, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		return body, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}

// Post performs a POST request (no retries - not idempotent)
func (c *HTTPClient) Post(url string, body io.Reader, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}
