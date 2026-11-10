package service

import (
	"context"
	"errors"
	"testing"

	"notequizai/internal/models"
	"notequizai/internal/storage"
)

type fakeGen struct{ topics int }

func (f *fakeGen) Generate(_ context.Context, topics []models.Topic, _ []models.QuestionSpec) (*models.Assessment, error) {
	f.topics = len(topics)
	return &models.Assessment{ID: "id1"}, nil
}

func TestCreateStoresAssessment(t *testing.T) {
	gen := &fakeGen{}
	svc := New(gen, storage.NewMemory(10))
	req := models.GenerateRequest{
		Notes:     "Some study notes",
		Questions: []models.QuestionSpec{{Type: models.OneMark, Count: 2}},
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil || a.ID != "id1" {
		t.Fatalf("unexpected result: %v, %v", a, err)
	}
	if gen.topics < 1 {
		t.Fatal("generator should have received topics")
	}
	if _, err := svc.Get(context.Background(), "id1"); err != nil {
		t.Fatalf("assessment should be stored: %v", err)
	}
}

func TestCreateRejectsInvalidRequest(t *testing.T) {
	gen := &fakeGen{}
	svc := New(gen, storage.NewMemory(10))

	_, err := svc.Create(context.Background(), models.GenerateRequest{Notes: "  "})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
	if gen.topics != 0 {
		t.Fatal("generator must not be called for invalid requests")
	}
}
