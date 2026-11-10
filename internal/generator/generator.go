package generator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"notequizai/internal/llm"
	"notequizai/internal/models"
	"notequizai/internal/validator"
)

type Generator struct {
	client      llm.Client
	workers     int
	maxAttempts int 
}

func New(client llm.Client, workers, maxAttempts int) *Generator {
	if workers < 1 {
		workers = 1
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &Generator{client: client, workers: workers, maxAttempts: maxAttempts}
}

func (g *Generator) Generate(ctx context.Context, topics []models.Topic, specs []models.QuestionSpec) (*models.Assessment, error) {
	start := time.Now()

	tasks := buildTasks(topics, specs)
	if len(tasks) == 0 {
		return nil, errors.New("nothing to generate: need at least one topic and one question spec")
	}

	results := runPool(ctx, tasks, g.workers, g.runTask)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	a, err := aggregate(results)
	if err != nil {
		return nil, err
	}
	a.ID = newID()
	a.CreatedAt = time.Now().UTC()
	a.GeneratedInMs = time.Since(start).Milliseconds()
	return a, nil
}

func (g *Generator) runTask(ctx context.Context, t Task) ([]models.Question, error) {
	prompt := llm.BuildPrompt(t.Topic, t.Type, t.Count)

	var lastErr error
	for attempt := 1; attempt <= g.maxAttempts; attempt++ {
		p := prompt
		if lastErr != nil {
			p += fmt.Sprintf("\n\nYour previous reply was rejected: %v\nReturn ONLY corrected JSON in the required format.", lastErr)
		}

		raw, err := g.client.Generate(ctx, p)
		if err != nil {
			return nil, err 
		}
		qs, err := validator.ParseQuestions(raw, t.Type, t.Count)
		if err == nil {
			return qs, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("model output still invalid after %d attempts: %w", g.maxAttempts, lastErr)
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
