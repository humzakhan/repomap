package analyzer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Task represents a single unit of work for the worker pool.
type Task struct {
	ID      string
	Request CompletionRequest
}

// TaskResult contains the outcome of a single task.
type TaskResult struct {
	TaskID       string
	Response     *CompletionResponse
	Error        error
	Retries      int
	Duration     time.Duration
}

// ProgressFunc is called after each task completes.
type ProgressFunc func(completed, total int, result TaskResult)

// Pool manages concurrent LLM API calls with bounded concurrency.
type Pool struct {
	concurrency        int
	provider           Provider
	maxRetries         int
	maxRateLimitWaits  int

	// Shared rate-limit backoff: when any worker hits a 429, all workers
	// wait until this time before sending new requests.
	rateLimitMu    sync.Mutex
	rateLimitUntil time.Time
}

// NewPool creates a worker pool with the given concurrency limit.
func NewPool(concurrency int, provider Provider) *Pool {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &Pool{
		concurrency:       concurrency,
		provider:          provider,
		maxRetries:        3,
		maxRateLimitWaits: 10,
	}
}

// Run executes all tasks concurrently with bounded parallelism.
// The progress callback is called after each task completes (thread-safe).
func (p *Pool) Run(ctx context.Context, tasks []Task, progress ProgressFunc) []TaskResult {
	results := make([]TaskResult, len(tasks))
	taskCh := make(chan int, len(tasks))

	// Feed task indices into the channel
	for i := range tasks {
		taskCh <- i
	}
	close(taskCh)

	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0

	for w := 0; w < p.concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range taskCh {
				select {
				case <-ctx.Done():
					mu.Lock()
					results[idx] = TaskResult{
						TaskID: tasks[idx].ID,
						Error:  ctx.Err(),
					}
					mu.Unlock()
					continue
				default:
				}

				result := p.executeWithRetry(ctx, tasks[idx])

				mu.Lock()
				results[idx] = result
				completed++
				if progress != nil {
					progress(completed, len(tasks), result)
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

// waitForRateLimit blocks until the shared rate-limit window has passed.
// Returns false if the context was cancelled while waiting.
func (p *Pool) waitForRateLimit(ctx context.Context) bool {
	p.rateLimitMu.Lock()
	waitUntil := p.rateLimitUntil
	p.rateLimitMu.Unlock()

	delay := time.Until(waitUntil)
	if delay <= 0 {
		return true
	}

	select {
	case <-time.After(delay):
		return true
	case <-ctx.Done():
		return false
	}
}

// setRateLimitBackoff updates the shared backoff window so all workers pause.
func (p *Pool) setRateLimitBackoff(d time.Duration) {
	p.rateLimitMu.Lock()
	defer p.rateLimitMu.Unlock()

	newUntil := time.Now().Add(d)
	if newUntil.After(p.rateLimitUntil) {
		p.rateLimitUntil = newUntil
	}
}

func (p *Pool) executeWithRetry(ctx context.Context, task Task) TaskResult {
	var lastErr error
	start := time.Now()
	rateLimitWaits := 0

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return TaskResult{
				TaskID:   task.ID,
				Error:    ctx.Err(),
				Retries:  attempt,
				Duration: time.Since(start),
			}
		default:
		}

		// Wait for any active rate-limit backoff before making a request
		if !p.waitForRateLimit(ctx) {
			return TaskResult{
				TaskID:   task.ID,
				Error:    ctx.Err(),
				Retries:  attempt,
				Duration: time.Since(start),
			}
		}

		resp, err := p.provider.Complete(ctx, task.Request)
		if err == nil {
			return TaskResult{
				TaskID:   task.ID,
				Response: resp,
				Retries:  attempt,
				Duration: time.Since(start),
			}
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Rate limit errors: wait for the requested duration and don't count
		// against normal retries.
		var rlErr *RateLimitError
		if errors.As(err, &rlErr) {
			rateLimitWaits++
			if rateLimitWaits > p.maxRateLimitWaits {
				return TaskResult{
					TaskID:   task.ID,
					Error:    fmt.Errorf("rate limited %d times, giving up: %w", rateLimitWaits, lastErr),
					Retries:  attempt,
					Duration: time.Since(start),
				}
			}

			// Tell all workers to back off
			p.setRateLimitBackoff(rlErr.RetryAfter)

			// Don't count this attempt against maxRetries
			attempt--
			continue
		}

		// Non-rate-limit error: exponential backoff with jitter
		if attempt < p.maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			select {
			case <-time.After(backoff + jitter):
			case <-ctx.Done():
				return TaskResult{
					TaskID:   task.ID,
					Error:    ctx.Err(),
					Retries:  attempt + 1,
					Duration: time.Since(start),
				}
			}
		}
	}

	return TaskResult{
		TaskID:   task.ID,
		Error:    fmt.Errorf("all %d retries exhausted: %w", p.maxRetries, lastErr),
		Retries:  p.maxRetries,
		Duration: time.Since(start),
	}
}
