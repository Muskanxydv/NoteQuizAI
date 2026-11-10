package storage

import (
	"context"
	"errors"
	"testing"

	"notequizai/internal/models"
)

func TestMemorySaveAndGet(t *testing.T) {
	m := NewMemory(10)
	ctx := context.Background()

	if err := m.Save(ctx, &models.Assessment{ID: "a"}); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(ctx, "a")
	if err != nil || got.ID != "a" {
		t.Fatalf("expected assessment a, got %v, %v", got, err)
	}
	if _, err := m.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryEvictsOldest(t *testing.T) {
	m := NewMemory(2)
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		m.Save(ctx, &models.Assessment{ID: id})
	}
	if _, err := m.Get(ctx, "a"); !errors.Is(err, ErrNotFound) {
		t.Fatal("oldest assessment should have been evicted")
	}
	if _, err := m.Get(ctx, "c"); err != nil {
		t.Fatal("newest assessment should still be stored")
	}
}
