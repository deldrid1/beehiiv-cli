package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/deldrid1/beehiiv-cli/internal/commandset"
	"github.com/deldrid1/beehiiv-cli/internal/config"
	"github.com/deldrid1/beehiiv-cli/internal/ratelimit"
)

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type Error struct {
	Operation string          `json:"operation"`
	Status    int             `json:"status"`
	Message   string          `json:"message"`
	Body      json.RawMessage `json:"body,omitempty"`
}

func (e *Error) Error() string {
	if e.Status == 0 {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("%s failed with status %d: %s", e.Operation, e.Status, e.Message)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL       string
	apiKey        string
	publicationID string
	httpClient    HTTPClient
	limiter       *ratelimit.Limiter
	timeout       time.Duration
	debug         bool
	verbose       bool
	debugWriter   io.Writer
}

func New(runtime config.Runtime, httpClient HTTPClient, debugWriter io.Writer) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if debugWriter == nil {
		debugWriter = os.Stderr
	}
	return &Client{
		baseURL:       strings.TrimRight(runtime.BaseURL, "/"),
		apiKey:        runtime.APIKey,
		publicationID: runtime.PublicationID,
		httpClient:    httpClient,
		limiter:       ratelimit.New(runtime.RateLimitRPM),
		timeout:       runtime.Timeout,
		debug:         runtime.Debug,
		verbose:       runtime.Verbose,
		debugWriter:   debugWriter,
	}
}

func (c *Client) Execute(
	ctx context.Context,
	operation commandset.Operation,
	pathValues map[string]string,
	query url.Values,
	body []byte,
) (*Response, error) {
	requestURL, err := c.buildURL(operation, pathValues, query)
	if err != nil {
		return nil, &Error{Operation: operation.OperationID, Message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, &Error{Operation: operation.OperationID, Message: err.Error()}
		}

		requestBody := io.Reader(http.NoBody)
		if len(body) > 0 {
			requestBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, operation.Method, requestURL, requestBody)
		if err != nil {
			return nil, &Error{Operation: operation.OperationID, Message: err.Error()}
		}
		req.Header.Set("Accept", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		if c.debug {
			fmt.Fprintf(c.debugWriter, "%s %s\n", operation.Method, requestURL)
		}
		if c.verbose {
			c.dumpRequest(req, body)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, &Error{Operation: operation.OperationID, Message: err.Error()}
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, &Error{Operation: operation.OperationID, Status: resp.StatusCode, Message: readErr.Error()}
		}
		if c.verbose {
			c.dumpResponse(resp.Status, resp.Header, data)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			delay := c.limiter.RetryAfter(resp.Header, time.Now(), time.Second)
			if err := sleepContext(ctx, delay); err != nil {
				return nil, &Error{Operation: operation.OperationID, Status: resp.StatusCode, Message: err.Error(), Body: json.RawMessage(data)}
			}
			continue
		}

		c.limiter.Observe(resp.Header, time.Now())

		if resp.StatusCode >= 400 {
			return nil, &Error{
				Operation: operation.OperationID,
				Status:    resp.StatusCode,
				Message:   messageFromErrorBody(data, resp.Status),
				Body:      json.RawMessage(data),
			}
		}

		return &Response{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header.Clone(),
			Body:       data,
		}, nil
	}

	return nil, &Error{Operation: operation.OperationID, Message: "retry loop exhausted"}
}

func (c *Client) buildURL(operation commandset.Operation, pathValues map[string]string, query url.Values) (string, error) {
	path := operation.Path
	if operation.RequiresPublicationID {
		if c.publicationID == "" {
			return "", fmt.Errorf("publication id is required for %s", strings.Join(operation.Command, " "))
		}
		path = strings.ReplaceAll(path, "{publicationId}", url.PathEscape(c.publicationID))
	}

	for _, pathParam := range operation.PathParams {
		value := strings.TrimSpace(pathValues[pathParam])
		if value == "" {
			return "", fmt.Errorf("missing required path parameter %q", pathParam)
		}
		path = strings.ReplaceAll(path, "{"+pathParam+"}", url.PathEscape(value))
	}

	requestURL := c.baseURL + path
	if len(query) == 0 {
		return requestURL, nil
	}
	return requestURL + "?" + query.Encode(), nil
}

func (c *Client) dumpRequest(req *http.Request, body []byte) {
	fmt.Fprintf(c.debugWriter, "> %s %s\n", req.Method, req.URL.String())
	c.dumpHeaders(req.Header, true)
	if len(body) > 0 {
		fmt.Fprintln(c.debugWriter)
		fmt.Fprintln(c.debugWriter, string(body))
	}
}

func (c *Client) dumpResponse(status string, headers http.Header, body []byte) {
	fmt.Fprintf(c.debugWriter, "< %s\n", status)
	c.dumpHeaders(headers, false)
	if len(body) > 0 {
		fmt.Fprintln(c.debugWriter)
		fmt.Fprintln(c.debugWriter, string(body))
	}
}

func (c *Client) dumpHeaders(headers http.Header, redactAuth bool) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range headers.Values(key) {
			if redactAuth && strings.EqualFold(key, "Authorization") {
				value = redactAuthorization(value)
			}
			fmt.Fprintf(c.debugWriter, "%s: %s\n", key, value)
		}
	}
}

func redactAuthorization(value string) string {
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return "[REDACTED]"
	}
	return parts[0] + " [REDACTED]"
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func messageFromErrorBody(body []byte, fallback string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		if len(body) == 0 {
			return fallback
		}
		return string(body)
	}

	for _, key := range []string{"message", "error", "detail"} {
		if value, ok := payload[key]; ok {
			return fmt.Sprint(value)
		}
	}

	if errorsValue, ok := payload["errors"]; ok {
		return fmt.Sprint(errorsValue)
	}

	return fallback
}
