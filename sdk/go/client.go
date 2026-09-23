// Package faultwing sends errors from Go applications to a Faultwing project.
package faultwing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// Options sets monitoring context and the maximum time spent sending an event.
type Options struct {
	Environment string
	Release     string
	Timeout     time.Duration
}

// Event supplies a stable error type and location when automatic capture is unsuitable.
type Event struct {
	ExceptionType string
	Message       string
	Stacktrace    string
}

// Client sends events for one Faultwing project. It is safe for concurrent use.
type Client struct {
	eventsURL   string
	apiKey      string
	environment string
	release     string
	httpClient  *http.Client
}

// NewClient configures a project client. A zero Timeout uses two seconds.
func NewClient(baseURL, apiKey string, options Options) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("url must be an HTTP or HTTPS base URL without credentials, query, or fragment")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("api key is required")
	}
	if options.Timeout < 0 {
		return nil, errors.New("timeout must not be negative")
	}

	environment := strings.TrimSpace(options.Environment)
	if environment == "" {
		environment = "production"
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v1/events"
	parsed.RawPath = ""

	return &Client{
		eventsURL:   parsed.String(),
		apiKey:      apiKey,
		environment: environment,
		release:     strings.TrimSpace(options.Release),
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

// CaptureError sends an error using its concrete type and this call site for grouping.
// It returns false when the error is nil or delivery fails.
func (c *Client) CaptureError(ctx context.Context, cause error) bool {
	if cause == nil {
		return false
	}
	value := reflect.ValueOf(cause)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return false
	}
	errorType := reflect.TypeOf(cause)
	for errorType.Kind() == reflect.Pointer {
		errorType = errorType.Elem()
	}

	var location string
	if pc, file, line, ok := runtime.Caller(1); ok {
		function := runtime.FuncForPC(pc)
		if function != nil {
			location = function.Name() + " "
		}
		location += fmt.Sprintf("(%s:%d)", filepath.Base(file), line)
	}

	message := cause.Error()
	if message == "" {
		message = errorType.String()
	}
	return c.CaptureEvent(ctx, Event{
		ExceptionType: errorType.String(),
		Message:       message,
		Stacktrace:    location,
	})
}

// CaptureEvent sends an event and reports whether Faultwing accepted it.
// Acceptance means the event was queued; it does not wait for worker processing.
func (c *Client) CaptureEvent(ctx context.Context, event Event) bool {
	event.ExceptionType = strings.TrimSpace(event.ExceptionType)
	event.Message = strings.TrimSpace(event.Message)
	if event.ExceptionType == "" || event.Message == "" {
		return false
	}

	payload := struct {
		ExceptionType string `json:"exception_type"`
		Message       string `json:"message"`
		Stacktrace    string `json:"stacktrace"`
		Environment   string `json:"environment"`
		Release       string `json:"release,omitempty"`
	}{event.ExceptionType, event.Message, event.Stacktrace, c.environment, c.release}
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.eventsURL, bytes.NewReader(body))
	if err != nil {
		return false
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "faultwing-go/0.1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	return response.StatusCode >= 200 && response.StatusCode < 300
}
