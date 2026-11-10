package storage

import (
	"context"
	"sync"

	"notequizai/internal/models"
)

// Memory is a free, in-process store. It keeps the newest `max` assessments
// and evicts the oldest, so memory use stays bounded. Data is lost on restart.
type Memory struct {
	mu    sync.RWMutex
	items map[string]*models.Assessment
	order []string
	max   int
}

var _ Repository = (*Memory)(nil)

func NewMemory(max int) *Memory {
	if max < 1 {
		max = 1
	}
	return &Memory{items: make(map[string]*models.Assessment), max: max}
}

func (m *Memory) Save(_ context.Context, a *models.Assessment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.items[a.ID]; !exists {
		m.order = append(m.order, a.ID)
		if len(m.order) > m.max {
			delete(m.items, m.order[0])
			m.order = m.order[1:]
		}
	}
	m.items[a.ID] = a
	return nil
}

func (m *Memory) Get(_ context.Context, id string) (*models.Assessment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	a, ok := m.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}
