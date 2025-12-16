package testutil

import (
	"context"
	"errors"
	"time"
)

// CancelResult holds the result of running a function with cancellation.
type CancelResult struct {
	// Err is the error returned by the function (may be nil).
	Err error
	// WasCancelled is true if the error is context.Canceled.
	WasCancelled bool
	// Completed is true if the function returned before the timeout.
	Completed bool
	// Duration is how long the function ran.
	Duration time.Duration
}

// RunWithCancel runs a function with a cancellable context.
// It cancels the context after cancelAfter duration and waits up to
// timeout for the function to return.
//
// Example:
//
//	result := testutil.RunWithCancel(func(ctx context.Context) error {
//	    return myLongRunningFunction(ctx)
//	}, 50*time.Millisecond, 1*time.Second)
//
//	if !result.WasCancelled {
//	    t.Error("function did not respect cancellation")
//	}
func RunWithCancel(fn func(context.Context) error, cancelAfter, timeout time.Duration) CancelResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	errCh := make(chan error, 1)

	go func() {
		errCh <- fn(ctx)
	}()

	// Wait for cancelAfter, then cancel
	time.Sleep(cancelAfter)
	cancel()

	// Wait for function to return or timeout
	select {
	case err := <-errCh:
		return CancelResult{
			Err:          err,
			WasCancelled: errors.Is(err, context.Canceled),
			Completed:    true,
			Duration:     time.Since(start),
		}
	case <-time.After(timeout):
		return CancelResult{
			Err:          nil,
			WasCancelled: false,
			Completed:    false,
			Duration:     time.Since(start),
		}
	}
}

// RunWithTimeout runs a function with a timeout context.
// Returns the result and whether the function completed before timeout.
//
// Example:
//
//	result := testutil.RunWithTimeout(func(ctx context.Context) error {
//	    return myFunction(ctx)
//	}, 100*time.Millisecond)
//
//	if !result.Completed {
//	    t.Error("function timed out")
//	}
func RunWithTimeout(fn func(context.Context) error, timeout time.Duration) CancelResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	errCh := make(chan error, 1)

	go func() {
		errCh <- fn(ctx)
	}()

	select {
	case err := <-errCh:
		return CancelResult{
			Err:          err,
			WasCancelled: errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
			Completed:    true,
			Duration:     time.Since(start),
		}
	case <-time.After(timeout + 100*time.Millisecond): // Small buffer
		return CancelResult{
			Err:          nil,
			WasCancelled: false,
			Completed:    false,
			Duration:     time.Since(start),
		}
	}
}

// WaitForCondition polls a condition function until it returns true or timeout.
// Useful for waiting for async state changes in tests.
//
// Example:
//
//	ok := testutil.WaitForCondition(func() bool {
//	    return server.IsReady()
//	}, 100*time.Millisecond, 2*time.Second)
//
//	if !ok {
//	    t.Error("server did not become ready")
//	}
func WaitForCondition(condition func() bool, pollInterval, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(pollInterval)
	}
	return false
}
