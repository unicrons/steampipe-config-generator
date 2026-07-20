package aws

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestFetchConcurrently_AllSucceed(t *testing.T) {
	const n = 10
	var calls int32

	err := fetchConcurrently(t.Context(), n, 3, func(ctx context.Context, i int) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != n {
		t.Errorf("calls = %d, want %d", got, n)
	}
}

// TestFetchConcurrently_OneFailureIsNotSilenced is the regression test for the bug this
// helper replaces: previously, an error from a single item's fetch was logged and dropped,
// leaving that item silently incomplete instead of failing the whole batch.
func TestFetchConcurrently_OneFailureIsNotSilenced(t *testing.T) {
	wantErr := errors.New("boom")

	err := fetchConcurrently(t.Context(), 10, 3, func(ctx context.Context, i int) error {
		if i == 7 {
			return wantErr
		}
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestFetchConcurrently_RespectsLimit(t *testing.T) {
	const limit = 3
	var current, max int32

	err := fetchConcurrently(t.Context(), 20, limit, func(ctx context.Context, i int) error {
		c := atomic.AddInt32(&current, 1)
		defer atomic.AddInt32(&current, -1)

		for {
			m := atomic.LoadInt32(&max)
			if c <= m || atomic.CompareAndSwapInt32(&max, m, c) {
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if max > limit {
		t.Errorf("observed %d concurrent calls, want <= %d", max, limit)
	}
}
