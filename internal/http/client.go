package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{
		client: &http.Client{Timeout: timeout},
	}
}

var defaultClient = NewHTTPClient(10 * time.Second)

func (c *HTTPClient) Request(method, url string, body interface{}, headers map[string]string, target interface{}) error {
	var reqBody io.Reader

	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("[gohan http] failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("[gohan http] failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, val := range headers {
		req.Header.Set(key, val)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("[gohan http] request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[gohan http] failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("[gohan http] error response %d: %s", resp.StatusCode, string(respBytes))
	}

	if target != nil {
		if err := json.Unmarshal(respBytes, target); err != nil {
			return fmt.Errorf("[gohan http] failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func FetchJSON(url string, target interface{}) error {
	return defaultClient.Request(http.MethodGet, url, nil, nil, target)
}

func PostJSON(url string, body interface{}, target interface{}) error {
	return defaultClient.Request(http.MethodPost, url, body, nil, target)
}