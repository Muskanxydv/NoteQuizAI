package generator

import (
	"context"
	"sync"

	"notequizai/internal/models"
)

type taskFunc func(ctx context.Context, t Task) ([]models.Question, error)

func runPool(ctx context.Context, tasks []Task, workers int, fn taskFunc) []Result {
	results := make([]Result, len(tasks))
	if workers > len(tasks) {
		workers = len(tasks)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan Task)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				if err := ctx.Err(); err != nil { // cancelled: skip remaining work quickly
					results[t.Index] = Result{Task: t, Err: err}
					continue
				}
				qs, err := fn(ctx, t)
				results[t.Index] = Result{Task: t, Questions: qs, Err: err}
			}
		}()
	}

	for _, t := range tasks {
		jobs <- t
	}
	close(jobs)
	wg.Wait()
	return results
}
