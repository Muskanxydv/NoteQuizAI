package models

// QuestionType is the assessment format a question belongs to.
type QuestionType string

const (
	OneMark QuestionType = "1-mark"  // objective, one-word / one-line answer
	TwoMark QuestionType = "2-mark"  // short answer
	TenMark QuestionType = "10-mark" // comprehensive explanation
)

// Valid reports whether t is a supported question type.
func (t QuestionType) Valid() bool {
	switch t {
	case OneMark, TwoMark, TenMark:
		return true
	}
	return false
}

// Marks returns the marks a question of this type carries.
func (t QuestionType) Marks() int {
	switch t {
	case OneMark:
		return 1
	case TwoMark:
		return 2
	case TenMark:
		return 10
	}
	return 0
}

// Question is a single generated question. The JSON tags define the exact
// shape the LLM is instructed to return and that the validator enforces.
type Question struct {
	Question  string   `json:"question"`
	Answer    string   `json:"answer"`
	KeyPoints []string `json:"key_points,omitempty"` // marking points, used for 10-mark answers
}
