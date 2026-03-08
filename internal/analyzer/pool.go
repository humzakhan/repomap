package analyzer

import (
	"context"
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
	concurrency int
	provider    Provider
	maxRetries  int
}

// NewPool creates a worker pool with the given concurrency limit.
func NewPool(concurrency int, provider Provider) *Pool {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &Pool{
		concurrency: concurrency,
		provider:    provider,
		maxRetries:  3,
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

func (p *Pool) executeWithRetry(ctx context.Context, task Task) TaskResult {
	var lastErr error
	start := time.Now()

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

		// Exponential backoff with jitter
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
