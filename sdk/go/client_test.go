package faultwing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type receivedEvent struct {
	ExceptionType string `json:"exception_type"`
	Message       string `json:"message"`
	Stacktrace    string `json:"stacktrace"`
	Environment   string `json:"environment"`
	Release       string `json:"release"`
}

func TestCaptureEventSendsProjectEvent(t *testing.T) {
	type requestData struct {
		path, authorization, contentType string
		event                            receivedEvent
	}
	requests := make(chan requestData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event receivedEvent
		_ = json.NewDecoder(r.Body).Decode(&event)
		requests <- requestData{r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"), event}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/", "faultwing_test-key", Options{
		Environment: "development", Release: "1.3.2", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !client.CaptureEvent(context.Background(), Event{
		ExceptionType: "DatabaseTimeoutError",
		Message:       "connection timed out",
		Stacktrace:    "db/client.go:42",
	}) {
		t.Fatal("expected the event to be accepted")
	}

	got := <-requests
	if got.path != "/api/v1/events" || got.authorization != "Bearer faultwing_test-key" || got.contentType != "application/json" {
		t.Fatalf("unexpected request metadata: %+v", got)
	}
	if got.event != (receivedEvent{"DatabaseTimeoutError", "connection timed out", "db/client.go:42", "development", "1.3.2"}) {
		t.Fatalf("unexpected event: %+v", got.event)
	}
}

type databaseTimeoutError struct{}

func (databaseTimeoutError) Error() string { return "connection timed out" }

func TestCaptureErrorUsesStableCallerLocation(t *testing.T) {
	events := make(chan receivedEvent, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event receivedEvent
		_ = json.NewDecoder(r.Body).Decode(&event)
		events <- event
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "faultwing_test-key", Options{})
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if !client.CaptureError(context.Background(), databaseTimeoutError{}) {
			t.Fatal("expected the error to be accepted")
		}
	}
	first, second := <-events, <-events
	if first.ExceptionType != "faultwing.databaseTimeoutError" || first.Message != "connection timed out" {
		t.Fatalf("unexpected captured error: %+v", first)
	}
	if first.Stacktrace == "" || first.Stacktrace != second.Stacktrace || !strings.Contains(first.Stacktrace, "client_test.go:") {
		t.Fatalf("expected a stable caller location, got %q and %q", first.Stacktrace, second.Stacktrace)
	}
}

func TestCaptureEventReturnsFalseWhenDeliveryFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	client, err := NewClient(server.URL, "faultwing_test-key", Options{})
	if err != nil {
		t.Fatal(err)
	}
	event := Event{ExceptionType: "Error", Message: "broken"}
	if client.CaptureEvent(context.Background(), event) {
		t.Fatal("expected false for a rejected request")
	}
	server.Close()
	if client.CaptureEvent(context.Background(), event) {
		t.Fatal("expected false for a connection failure")
	}
}
