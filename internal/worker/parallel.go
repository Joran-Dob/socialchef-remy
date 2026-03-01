package worker

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ParallelFunc is a function that can be executed in parallel.
type ParallelFunc func(ctx context.Context) error

// ParallelResult holds the results from parallel operations.
type ParallelResult struct {
	Errors []error
}

// RunParallel executes multiple functions concurrently and returns when all complete.
// If any function returns an error, it is collected but execution continues.
// Context cancellation is respected - all goroutines will be cancelled if the context is cancelled.
func RunParallel(ctx context.Context, funcs []ParallelFunc) ParallelResult {
	if len(funcs) == 0 {
		return ParallelResult{}
	}

	// Use errgroup for proper error handling and context cancellation
	g, ctx := errgroup.WithContext(ctx)
	errors := make([]error, len(funcs))
	var mu sync.Mutex

	for i, fn := range funcs {
		i, fn := i, fn // capture loop variables
		g.Go(func() error {
			if err := fn(ctx); err != nil {
				mu.Lock()
				errors[i] = err
				mu.Unlock()
			}
			return nil // errgroup stops on first error, so we always return nil
		})
	}

	// Wait for all goroutines to complete
	_ = g.Wait()

	// Filter out nil errors
	var nonNilErrors []error
	for _, err := range errors {
		if err != nil {
			nonNilErrors = append(nonNilErrors, err)
		}
	}

	return ParallelResult{Errors: nonNilErrors}
}

// RunParallelWithResults executes multiple functions concurrently and collects their results.
// The results slice must be pre-allocated with the same length as funcs.
func RunParallelWithResults[T any](ctx context.Context, funcs []func(ctx context.Context) (T, error)) ([]T, []error) {
	if len(funcs) == 0 {
		return nil, nil
	}

	results := make([]T, len(funcs))
	errors := make([]error, len(funcs))

	var wg sync.WaitGroup
	wg.Add(len(funcs))

	for i, fn := range funcs {
		i, fn := i, fn // capture loop variables
		go func() {
			defer wg.Done()
			result, err := fn(ctx)
			results[i] = result
			errors[i] = err
		}()
	}

	wg.Wait()

	// Filter out nil errors
	var nonNilErrors []error
	for _, err := range errors {
		if err != nil {
			nonNilErrors = append(nonNilErrors, err)
		}
	}

	return results, nonNilErrors
}
