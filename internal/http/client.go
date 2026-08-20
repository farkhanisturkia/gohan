package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	netHttp "net/http"
	"strings"
	"time"
)

type HTTPClient struct {
	client *netHttp.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{
		client: &netHttp.Client{Timeout: timeout},
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

	req, err := netHttp.NewRequest(method, url, reqBody)
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
		if strTarget, ok := target.(*string); ok {
			*strTarget = string(respBytes)
			return nil
		}

		if err := json.Unmarshal(respBytes, target); err != nil {
			preview := string(respBytes)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			return fmt.Errorf("[gohan http] failed to unmarshal response: %w (raw response: %q)", err, strings.TrimSpace(preview))
		}
	}

	return nil
}

func FetchJSON(url string, target interface{}) error {
	return defaultClient.Request(netHttp.MethodGet, url, nil, nil, target)
}

func PostJSON(url string, body interface{}, target interface{}) error {
	return defaultClient.Request(netHttp.MethodPost, url, body, nil, target)
}