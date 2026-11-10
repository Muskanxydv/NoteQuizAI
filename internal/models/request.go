package models

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxNotesLength  = 50_000 // characters
	MaxCountPerType = 20
)

// QuestionSpec asks for Count questions of a given Type per topic.
type QuestionSpec struct {
	Type  QuestionType `json:"type"`
	Count int          `json:"count"`
}

// GenerateRequest is the body of POST /api/v1/assessments.
type GenerateRequest struct {
	Notes     string         `json:"notes"`
	Questions []QuestionSpec `json:"questions"`
}

// Validate checks the request before any LLM work is scheduled.
func (r *GenerateRequest) Validate() error {
	if strings.TrimSpace(r.Notes) == "" {
		return errors.New("notes must not be empty")
	}
	if len(r.Notes) > MaxNotesLength {
		return fmt.Errorf("notes exceed %d characters", MaxNotesLength)
	}
	if len(r.Questions) == 0 {
		return errors.New("at least one question spec is required")
	}
	seen := make(map[QuestionType]bool)
	for _, q := range r.Questions {
		if !q.Type.Valid() {
			return fmt.Errorf("unsupported question type %q", q.Type)
		}
		if seen[q.Type] {
			return fmt.Errorf("question type %s is listed more than once", q.Type)
		}
		seen[q.Type] = true
		if q.Count < 1 || q.Count > MaxCountPerType {
			return fmt.Errorf("count for %s must be between 1 and %d", q.Type, MaxCountPerType)
		}
	}
	return nil
}
