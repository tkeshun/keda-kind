package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	enqueueapp "keda-kind/sample-app/internal/enqueue"
)

type fakeTickService struct {
	result enqueueapp.TickResult
	err    error
	calls  int
}

func (f *fakeTickService) Tick(context.Context) (enqueueapp.TickResult, error) {
	f.calls++
	return f.result, f.err
}

func TestNewHTTPHandlerAcceptsEnqueueRequests(t *testing.T) {
	service := &fakeTickService{result: enqueueapp.TickResult{Sent: true}}

	handler := newHTTPHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/enqueue", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if service.calls != 1 {
		t.Fatalf("unexpected tick calls: %d", service.calls)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "accepted" {
		t.Fatalf("unexpected response status: %q", payload["status"])
	}
}

func TestNewHTTPHandlerReturnsBusyWhenEnqueueSkipped(t *testing.T) {
	service := &fakeTickService{result: enqueueapp.TickResult{Skipped: true}}

	handler := newHTTPHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/enqueue", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "skipped" {
		t.Fatalf("unexpected response status: %q", payload["status"])
	}
}

func TestNewHTTPHandlerReturnsInternalErrorWhenTickFails(t *testing.T) {
	service := &fakeTickService{err: errors.New("boom")}

	handler := newHTTPHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/enqueue", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "error" {
		t.Fatalf("unexpected response status: %q", payload["status"])
	}
}

func TestNewHTTPHandlerReportsHealth(t *testing.T) {
	handler := newHTTPHandler(&fakeTickService{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected response status: %q", payload["status"])
	}
}
