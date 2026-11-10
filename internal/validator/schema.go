// Package validator turns raw LLM text into trusted, typed questions.
// LLM output is untrusted input: it is parsed, checked against the rules for
// its question type, and rejected if it does not conform.
package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"notequizai/internal/models"
)

// ErrInvalid marks output that failed validation; callers may re-ask the model.
var ErrInvalid = errors.New("invalid model output")

const (
	maxOneMarkWords    = 12
	minTenMarkKeyPoint = 3
)

// ParseQuestions maps the model's JSON into []models.Question and validates it.
// Extra questions beyond `want` are dropped; fewer than `want` is an error.
func ParseQuestions(raw string, qt models.QuestionType, want int) ([]models.Question, error) {
	if !qt.Valid() {
		return nil, fmt.Errorf("%w: unknown question type %q", ErrInvalid, qt)
	}
	text := stripFences(raw)

	var qs []models.Question
	if strings.HasPrefix(text, "[") {
		if err := json.Unmarshal([]byte(text), &qs); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
	} else {
		var env struct {
			Questions []models.Question `json:"questions"`
		}
		if err := json.Unmarshal([]byte(text), &env); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		qs = env.Questions
	}

	if len(qs) < want {
		return nil, fmt.Errorf("%w: got %d questions, want %d", ErrInvalid, len(qs), want)
	}
	qs = qs[:want]

	for i := range qs {
		q := &qs[i]
		q.Question = strings.TrimSpace(q.Question)
		q.Answer = strings.TrimSpace(q.Answer)
		if q.Question == "" || q.Answer == "" {
			return nil, fmt.Errorf("%w: question %d is missing its question or answer", ErrInvalid, i+1)
		}
		switch qt {
		case models.OneMark:
			if n := len(strings.Fields(q.Answer)); n > maxOneMarkWords {
				return nil, fmt.Errorf("%w: question %d: 1-mark answer has %d words, max %d", ErrInvalid, i+1, n, maxOneMarkWords)
			}
			q.KeyPoints = nil
		case models.TwoMark:
			q.KeyPoints = nil
		case models.TenMark:
			q.KeyPoints = cleanPoints(q.KeyPoints)
			if len(q.KeyPoints) < minTenMarkKeyPoint {
				return nil, fmt.Errorf("%w: question %d: 10-mark answer needs at least %d key points", ErrInvalid, i+1, minTenMarkKeyPoint)
			}
		}
	}
	return qs, nil
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func cleanPoints(points []string) []string {
	var out []string
	for _, p := range points {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
