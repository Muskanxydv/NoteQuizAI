// Package service holds the business flow: validate -> split notes -> generate -> store.
package service

import (
	"context"
	"errors"
	"fmt"

	"notequizai/internal/models"
	"notequizai/internal/parser"
	"notequizai/internal/storage"
)

// ErrInvalidRequest marks problems caused by the caller's input (HTTP 400).
var ErrInvalidRequest = errors.New("invalid request")

// Generator is satisfied by *generator.Generator; an interface keeps this
// package easy to test.
type Generator interface {
	Generate(ctx context.Context, topics []models.Topic, specs []models.QuestionSpec) (*models.Assessment, error)
}

type Service struct {
	gen  Generator
	repo storage.Repository
}

func New(gen Generator, repo storage.Repository) *Service {
	return &Service{gen: gen, repo: repo}
}

func (s *Service) Create(ctx context.Context, req models.GenerateRequest) (*models.Assessment, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	topics := parser.Split(req.Notes)
	if len(topics) == 0 {
		return nil, fmt.Errorf("%w: no usable topics found in notes", ErrInvalidRequest)
	}

	a, err := s.gen.Generate(ctx, topics, req.Questions)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, fmt.Errorf("saving assessment: %w", err)
	}
	return a, nil
}

func (s *Service) Get(ctx context.Context, id string) (*models.Assessment, error) {
	return s.repo.Get(ctx, id)
}
