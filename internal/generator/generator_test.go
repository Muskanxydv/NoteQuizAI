package generator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"notequizai/internal/llm"
	"notequizai/internal/models"
)

const validResponse = `{"questions":[{"question":"Q1?","answer":"A1"},{"question":"Q2?","answer":"A2"},{"question":"Q3?","answer":"A3"}]}`

type fakeClient struct {
	delay   time.Duration
	calls   atomic.Int32
	respond func(call int, prompt string) (string, error)
}

func (f *fakeClient) Generate(ctx context.Context, prompt string) (string, error) {
	n := int(f.calls.Add(1))
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return f.respond(n, prompt)
}

func alwaysValid(int, string) (string, error) { return validResponse, nil }

var oneMark3 = []models.QuestionSpec{{Type: models.OneMark, Count: 3}}

func TestGenerateBuildsOrderedSections(t *testing.T) {
	g := New(&fakeClient{respond: alwaysValid}, 4, 2)
	topics := []models.Topic{{Title: "Alpha", Content: "a"}, {Title: "Beta", Content: "b"}}

	a, err := g.Generate(context.Background(), topics, oneMark3)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Sections) != 2 || a.Sections[0].Topic != "Alpha" || a.Sections[1].Topic != "Beta" {
		t.Fatalf("sections out of order: %+v", a.Sections)
	}
	if a.TotalMarks != 6 {
		t.Fatalf("expected 6 total marks, got %d", a.TotalMarks)
	}
	if a.ID == "" {
		t.Fatal("expected an assessment ID")
	}
}

func TestConcurrencyIsFasterThanSequential(t *testing.T) {
	topics := make([]models.Topic, 6)
	for i := range topics {
		topics[i] = models.Topic{Title: fmt.Sprintf("T%d", i), Content: "x"}
	}
	g := New(&fakeClient{delay: 150 * time.Millisecond, respond: alwaysValid}, 6, 2)

	start := time.Now()
	if _, err := g.Generate(context.Background(), topics, oneMark3); err != nil {
		t.Fatal(err)
	}
	// 6 tasks x 150ms = 900ms if sequential; concurrent should take ~150ms.
	if elapsed := time.Since(start); elapsed > 450*time.Millisecond {
		t.Fatalf("expected concurrent run under 450ms, took %v", elapsed)
	}
}

func TestPartialFailureReturnsWarnings(t *testing.T) {
	respond := func(_ int, prompt string) (string, error) {
		if strings.Contains(prompt, "Topic: Beta") {
			return "", errors.New("boom")
		}
		return validResponse, nil
	}
	g := New(&fakeClient{respond: respond}, 4, 2)
	topics := []models.Topic{{Title: "Alpha", Content: "a"}, {Title: "Beta", Content: "b"}}

	a, err := g.Generate(context.Background(), topics, oneMark3)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Sections) != 1 || len(a.Warnings) != 1 {
		t.Fatalf("expected 1 section and 1 warning, got %d and %d", len(a.Sections), len(a.Warnings))
	}
}

func TestInvalidJSONIsReAsked(t *testing.T) {
	fake := &fakeClient{respond: func(call int, _ string) (string, error) {
		if call == 1 {
			return "Sure! Here are your questions", nil
		}
		return validResponse, nil
	}}
	g := New(fake, 1, 2)

	a, err := g.Generate(context.Background(), []models.Topic{{Title: "Alpha", Content: "a"}}, oneMark3)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Sections) != 1 || fake.calls.Load() != 2 {
		t.Fatalf("expected 1 section after 2 calls, got %d sections, %d calls", len(a.Sections), fake.calls.Load())
	}
}

func TestAllTasksFailingReturnsError(t *testing.T) {
	fake := &fakeClient{respond: func(int, string) (string, error) { return "", llm.ErrDailyQuota }}
	g := New(fake, 2, 2)

	_, err := g.Generate(context.Background(), []models.Topic{{Title: "Alpha", Content: "a"}}, oneMark3)
	if !errors.Is(err, llm.ErrDailyQuota) {
		t.Fatalf("expected ErrDailyQuota, got %v", err)
	}
}
