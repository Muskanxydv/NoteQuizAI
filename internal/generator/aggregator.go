package generator

import (
	"fmt"

	"notequizai/internal/models"
)

func aggregate(results []Result) (*models.Assessment, error) {
	a := &models.Assessment{}
	var firstErr error
	failed := 0

	for _, r := range results {
		if r.Err != nil {
			failed++
			if firstErr == nil {
				firstErr = r.Err
			}
			a.Warnings = append(a.Warnings, fmt.Sprintf("%q (%s): %v", r.Task.Topic.Title, r.Task.Type, r.Err))
			continue
		}
		a.Sections = append(a.Sections, models.Section{
			Topic:     r.Task.Topic.Title,
			Type:      r.Task.Type,
			Marks:     r.Task.Type.Marks(),
			Questions: r.Questions,
		})
	}

	if failed == len(results) {
		return nil, fmt.Errorf("all %d generation tasks failed: %w", failed, firstErr)
	}
	a.ComputeTotalMarks()
	return a, nil
}
