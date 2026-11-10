package models

import "time"

// Section holds all questions of one type generated for one topic.
type Section struct {
	Topic     string       `json:"topic"`
	Type      QuestionType `json:"type"`
	Marks     int          `json:"marks_each"`
	Questions []Question   `json:"questions"`
}

// Assessment is the final aggregated result returned to the client.
type Assessment struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TotalMarks    int       `json:"total_marks"`
	Sections      []Section `json:"sections"`
	GeneratedInMs int64     `json:"generated_in_ms"`    // wall-clock generation time
	Warnings      []string  `json:"warnings,omitempty"` // sections that could not be generated
}

// ComputeTotalMarks sums marks across all sections.
func (a *Assessment) ComputeTotalMarks() {
	total := 0
	for _, s := range a.Sections {
		total += s.Marks * len(s.Questions)
	}
	a.TotalMarks = total
}
