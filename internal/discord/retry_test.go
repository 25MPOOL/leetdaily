package discord

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryHTTPClientNoRetryOn200(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryHTTPClientNoRetryOn4xx(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("StatusCode = %d, want 403", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryHTTPClientRetriesOn5xxThenSucceeds(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestRetryHTTPClientExhaustsRetries(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error")) //nolint:errcheck
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  2,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("StatusCode = %d, want 500", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "error" {
		t.Fatalf("body = %q, want %q", body, "error")
	}
	// 1 initial + 2 retries = 3
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestRetryHTTPClientRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  5,
		InitialWait: 1 * time.Second,
		MaxWait:     5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	_, err := client.Do(req)

	if err == nil {
		t.Fatal("Do() error = nil, want context.Canceled")
	}
	if err != context.Canceled {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
	if calls.Load() < 1 {
		t.Fatal("expected at least 1 call before cancellation")
	}
}

func TestRetryHTTPClientRetriesOnNetworkError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	failCount := atomic.Int32{}
	failClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			n := failCount.Add(1)
			if n <= 2 {
				return nil, errors.New("connection refused")
			}
			return server.Client().Do(req)
		},
	}

	client := NewRetryHTTPClient(failClient, RetryOptions{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if failCount.Load() != 3 {
		t.Fatalf("attempts = %d, want 3", failCount.Load())
	}
}

func TestRetryHTTPClientRetriesPostWithBody(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var lastBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		lastBody = string(body)
		if n <= 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryHTTPClient(server.Client(), RetryOptions{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     50 * time.Millisecond,
	})

	payload := []byte(`{"name":"test thread"}`)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if lastBody != `{"name":"test thread"}` {
		t.Fatalf("lastBody = %q, want %q", lastBody, `{"name":"test thread"}`)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

type mockHTTPClient struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}
