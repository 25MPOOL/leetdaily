package discord

import (
	"context"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

type RetryOptions struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

type RetryHTTPClient struct {
	inner       HTTPClient
	maxRetries  int
	initialWait time.Duration
	maxWait     time.Duration
}

func NewRetryHTTPClient(inner HTTPClient, opts RetryOptions) *RetryHTTPClient {
	if inner == nil {
		inner = http.DefaultClient
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	initialWait := opts.InitialWait
	if initialWait <= 0 {
		initialWait = 1 * time.Second
	}
	maxWait := opts.MaxWait
	if maxWait <= 0 {
		maxWait = 30 * time.Second
	}

	return &RetryHTTPClient{
		inner:       inner,
		maxRetries:  maxRetries,
		initialWait: initialWait,
		maxWait:     maxWait,
	}
}

func (r *RetryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			wait := r.backoff(attempt - 1)
			if sleepErr := sleepContext(req.Context(), wait); sleepErr != nil {
				return nil, sleepErr
			}

			if req.GetBody != nil {
				body, bodyErr := req.GetBody()
				if bodyErr != nil {
					return nil, bodyErr
				}
				req.Body = body
			}
		}

		resp, err = r.inner.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode < 500 {
			return resp, nil
		}

		if attempt < r.maxRetries {
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
		}
	}

	return resp, nil
}

func (r *RetryHTTPClient) backoff(attempt int) time.Duration {
	base := float64(r.initialWait) * math.Pow(2, float64(attempt))
	jitter := time.Duration(rand.Int64N(int64(500 * time.Millisecond)))
	wait := time.Duration(base) + jitter
	if wait > r.maxWait {
		wait = r.maxWait
	}
	return wait
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
