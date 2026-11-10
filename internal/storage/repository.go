// Package storage holds generated assessments.
package storage

import (
	"context"
	"errors"

	"notequizai/internal/models"
)

var ErrNotFound = errors.New("assessment not found")

// Repository is the storage contract. Memory implements it today; a database
// version could replace it without touching any other package.
type Repository interface {
	Save(ctx context.Context, a *models.Assessment) error
	Get(ctx context.Context, id string) (*models.Assessment, error)
}
