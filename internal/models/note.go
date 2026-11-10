package models

// Topic is one logical chunk of the student's notes. Each topic becomes
// its own set of concurrent LLM tasks.
type Topic struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
