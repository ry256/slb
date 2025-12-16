package testutil

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunWithCancel_FunctionRespectsCancellation(t *testing.T) {
	// A function that respects cancellation
	fn := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	result := RunWithCancel(fn, 50*time.Millisecond, 1*time.Second)

	if !result.Completed {
		t.Error("function should have completed")
	}
	if !result.WasCancelled {
		t.Error("function should have been cancelled")
	}
	if !errors.Is(result.Err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", result.Err)
	}
}

func TestRunWithCancel_FunctionIgnoresCancellation(t *testing.T) {
	// A function that ignores cancellation (bad behavior)
	fn := func(ctx context.Context) error {
		time.Sleep(2 * time.Second) // Ignores context
		return nil
	}

	result := RunWithCancel(fn, 50*time.Millisecond, 200*time.Millisecond)

	if result.Completed {
		t.Error("function should not have completed (it ignores cancellation)")
	}
	if result.WasCancelled {
		t.Error("should not report cancelled if it didn't complete")
	}
}

func TestRunWithCancel_FunctionCompletesBeforeCancel(t *testing.T) {
	// A function that completes quickly
	fn := func(ctx context.Context) error {
		return nil // Returns immediately
	}

	result := RunWithCancel(fn, 100*time.Millisecond, 1*time.Second)

	if !result.Completed {
		t.Error("function should have completed")
	}
	if result.WasCancelled {
		t.Error("function completed before cancel, should not be marked cancelled")
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}
}

func TestRunWithCancel_FunctionReturnsError(t *testing.T) {
	expectedErr := errors.New("test error")
	fn := func(ctx context.Context) error {
		return expectedErr
	}

	result := RunWithCancel(fn, 100*time.Millisecond, 1*time.Second)

	if !result.Completed {
		t.Error("function should have completed")
	}
	if !errors.Is(result.Err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, result.Err)
	}
}

func TestRunWithTimeout_CompletesBeforeTimeout(t *testing.T) {
	fn := func(ctx context.Context) error {
		return nil
	}

	result := RunWithTimeout(fn, 100*time.Millisecond)

	if !result.Completed {
		t.Error("function should have completed")
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}
}

func TestRunWithTimeout_TimesOut(t *testing.T) {
	fn := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	}

	result := RunWithTimeout(fn, 50*time.Millisecond)

	if !result.Completed {
		t.Error("function should have completed (after timeout)")
	}
	if !result.WasCancelled {
		t.Error("should be marked as cancelled due to deadline")
	}
}

func TestWaitForCondition_ConditionTrue(t *testing.T) {
	counter := 0
	condition := func() bool {
		counter++
		return counter >= 3
	}

	ok := WaitForCondition(condition, 10*time.Millisecond, 1*time.Second)

	if !ok {
		t.Error("condition should have become true")
	}
	if counter < 3 {
		t.Errorf("condition should have been checked at least 3 times, got %d", counter)
	}
}

func TestWaitForCondition_ConditionNeverTrue(t *testing.T) {
	condition := func() bool {
		return false
	}

	ok := WaitForCondition(condition, 10*time.Millisecond, 50*time.Millisecond)

	if ok {
		t.Error("condition should have timed out")
	}
}

func TestWaitForCondition_ImmediatelyTrue(t *testing.T) {
	condition := func() bool {
		return true
	}

	start := time.Now()
	ok := WaitForCondition(condition, 10*time.Millisecond, 1*time.Second)
	duration := time.Since(start)

	if !ok {
		t.Error("condition should have been true")
	}
	if duration > 50*time.Millisecond {
		t.Errorf("should have returned immediately, took %v", duration)
	}
}

func TestCancelResult_StructFields(t *testing.T) {
	r := CancelResult{
		Err:          errors.New("test"),
		WasCancelled: true,
		Completed:    true,
		Duration:     100 * time.Millisecond,
	}

	if r.Err == nil {
		t.Error("Err should not be nil")
	}
	if !r.WasCancelled {
		t.Error("WasCancelled should be true")
	}
	if !r.Completed {
		t.Error("Completed should be true")
	}
	if r.Duration != 100*time.Millisecond {
		t.Errorf("Duration should be 100ms, got %v", r.Duration)
	}
}
