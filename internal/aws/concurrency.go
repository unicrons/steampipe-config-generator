package aws

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// fetchConcurrently calls fetch(ctx, i) for every i in [0, n), running at most limit calls at
// once. The first error returned by any call cancels ctx and is returned by fetchConcurrently -
// unlike the semaphore+WaitGroup loops this replaces, no error is ever dropped.
func fetchConcurrently(ctx context.Context, n, limit int, fetch func(ctx context.Context, i int) error) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(limit)

	for i := range n {
		group.Go(func() error { return fetch(ctx, i) })
	}

	return group.Wait()
}
