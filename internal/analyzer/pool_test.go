package analyzer

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type fakeProvider struct {
	callCount atomic.Int32
	failUntil int32 // fail this many times before succeeding
	delay     time.Duration
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	count := f.callCount.Add(1)
	if count <= f.failUntil {
		return nil, fmt.Errorf("transient error #%d", count)
	}

	return &CompletionResponse{
		Content:      `{"result": "ok"}`,
		InputTokens:  100,
		OutputTokens: 50,
		Model:        req.Model,
	}, nil
}

func (f *fakeProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	return float64(inputTokens+outputTokens) / 1_000_000
}

func TestPoolRunSingleTask(t *testing.T) {
	provider := &fakeProvider{}
	pool := NewPool(2, provider)

	tasks := []Task{
		{ID: "task-1", Request: CompletionRequest{Model: "test-model"}},
	}

	results := pool.Run(context.Background(), tasks, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error != nil {
		t.Fatalf("unexpected error: %v", results[0].Error)
	}
	if results[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", results[0].TaskID)
	}
	if results[0].Response.Content != `{"result": "ok"}` {
		t.Errorf("unexpected content: %s", results[0].Response.Content)
	}
}

func TestPoolRunMultipleTasks(t *testing.T) {
	provider := &fakeProvider{}
	pool := NewPool(3, provider)

	tasks := make([]Task, 5)
	for i := range tasks {
		tasks[i] = Task{
			ID:      fmt.Sprintf("task-%d", i),
			Request: CompletionRequest{Model: "test-model"},
		}
	}

	var completed atomic.Int32
	progress := func(c, total int, result TaskResult) {
		completed.Add(1)
	}

	results := pool.Run(context.Background(), tasks, progress)

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("task %d failed: %v", i, r.Error)
		}
	}

	if completed.Load() != 5 {
		t.Errorf("expected 5 progress calls, got %d", completed.Load())
	}
}

func TestPoolContextCancellation(t *testing.T) {
	provider := &fakeProvider{delay: 5 * time.Second}
	pool := NewPool(2, provider)

	tasks := []Task{
		{ID: "task-1", Request: CompletionRequest{Model: "test-model"}},
		{ID: "task-2", Request: CompletionRequest{Model: "test-model"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := pool.Run(ctx, tasks, nil)

	for _, r := range results {
		if r.Error == nil {
			t.Error("expected error due to context cancellation")
		}
	}
}

func TestPoolDefaultConcurrency(t *testing.T) {
	pool := NewPool(0, &fakeProvider{})
	if pool.concurrency != 10 {
		t.Errorf("expected default concurrency 10, got %d", pool.concurrency)
	}

	pool = NewPool(-1, &fakeProvider{})
	if pool.concurrency != 10 {
		t.Errorf("expected default concurrency 10, got %d", pool.concurrency)
	}
}

func TestPoolProgress(t *testing.T) {
	provider := &fakeProvider{}
	pool := NewPool(1, provider)

	tasks := []Task{
		{ID: "a", Request: CompletionRequest{Model: "m"}},
		{ID: "b", Request: CompletionRequest{Model: "m"}},
		{ID: "c", Request: CompletionRequest{Model: "m"}},
	}

	var progressCalls []int
	progress := func(completed, total int, result TaskResult) {
		progressCalls = append(progressCalls, completed)
	}

	pool.Run(context.Background(), tasks, progress)

	if len(progressCalls) != 3 {
		t.Fatalf("expected 3 progress calls, got %d", len(progressCalls))
	}

	// With concurrency 1, progress should be sequential
	for i, c := range progressCalls {
		if c != i+1 {
			t.Errorf("expected progress call %d to report %d completed, got %d", i, i+1, c)
		}
	}
}

// rateLimitProvider returns RateLimitError for the first N calls, then succeeds.
type rateLimitProvider struct {
	callCount    atomic.Int32
	rateLimitFor int32
	retryAfter   time.Duration
}

func (r *rateLimitProvider) Name() string { return "rate-limited" }

func (r *rateLimitProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	count := r.callCount.Add(1)
	if count <= r.rateLimitFor {
		return nil, &RateLimitError{
			RetryAfter: r.retryAfter,
			Message:    fmt.Sprintf("rate limited (call %d)", count),
		}
	}
	return &CompletionResponse{
		Content:      `{"result": "ok"}`,
		InputTokens:  100,
		OutputTokens: 50,
		Model:        req.Model,
	}, nil
}

func (r *rateLimitProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	return 0
}

func TestPoolRateLimitRetry(t *testing.T) {
	// Provider rate-limits the first 2 calls, then succeeds.
	// RetryAfter is very short so the test runs fast.
	provider := &rateLimitProvider{rateLimitFor: 2, retryAfter: 10 * time.Millisecond}
	pool := NewPool(1, provider)

	tasks := []Task{
		{ID: "task-1", Request: CompletionRequest{Model: "test-model"}},
	}

	results := pool.Run(context.Background(), tasks, nil)

	if results[0].Error != nil {
		t.Fatalf("expected success after rate-limit retries, got: %v", results[0].Error)
	}
	if results[0].Response.Content != `{"result": "ok"}` {
		t.Errorf("unexpected content: %s", results[0].Response.Content)
	}
	// Should not count rate-limit waits as retries
	if results[0].Retries != 0 {
		t.Errorf("expected 0 retries (rate limits don't count), got %d", results[0].Retries)
	}
}

func TestPoolRateLimitSharedBackoff(t *testing.T) {
	// Two concurrent workers: first call gets rate-limited, second worker
	// should wait for the shared backoff before attempting.
	provider := &rateLimitProvider{rateLimitFor: 1, retryAfter: 50 * time.Millisecond}
	pool := NewPool(2, provider)

	tasks := []Task{
		{ID: "task-1", Request: CompletionRequest{Model: "m"}},
		{ID: "task-2", Request: CompletionRequest{Model: "m"}},
	}

	start := time.Now()
	results := pool.Run(context.Background(), tasks, nil)
	elapsed := time.Since(start)

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("task %d failed: %v", i, r.Error)
		}
	}

	// Should have waited at least the retry-after duration
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected at least ~50ms for rate-limit backoff, got %v", elapsed)
	}
}

func TestPoolRateLimitExhaustion(t *testing.T) {
	// Provider always rate-limits — should give up after maxRateLimitWaits.
	provider := &rateLimitProvider{rateLimitFor: 100, retryAfter: 1 * time.Millisecond}
	pool := NewPool(1, provider)

	tasks := []Task{
		{ID: "task-1", Request: CompletionRequest{Model: "m"}},
	}

	results := pool.Run(context.Background(), tasks, nil)

	if results[0].Error == nil {
		t.Fatal("expected error after exhausting rate-limit retries")
	}
}
