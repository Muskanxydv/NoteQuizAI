package validator

import (
	"errors"
	"strings"
	"testing"

	"notequizai/internal/models"
)

func TestParseQuestions(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		qt      models.QuestionType
		want    int
		wantErr bool
	}{
		{"valid 1-mark", `{"questions":[{"question":"Q?","answer":"A"}]}`, models.OneMark, 1, false},
		{"code fences", "```json\n{\"questions\":[{\"question\":\"Q?\",\"answer\":\"A\"}]}\n```", models.OneMark, 1, false},
		{"bare array", `[{"question":"Q?","answer":"A"}]`, models.TwoMark, 1, false},
		{"extra questions dropped", `{"questions":[{"question":"Q1","answer":"A"},{"question":"Q2","answer":"B"}]}`, models.OneMark, 1, false},
		{"too few questions", `{"questions":[{"question":"Q?","answer":"A"}]}`, models.OneMark, 2, true},
		{"not json", `Sure! Here are your questions`, models.OneMark, 1, true},
		{"empty answer", `{"questions":[{"question":"Q?","answer":"  "}]}`, models.TwoMark, 1, true},
		{"1-mark answer too long", `{"questions":[{"question":"Q?","answer":"` + strings.Repeat("word ", 20) + `"}]}`, models.OneMark, 1, true},
		{"10-mark without key points", `{"questions":[{"question":"Q?","answer":"long answer"}]}`, models.TenMark, 1, true},
		{"valid 10-mark", `{"questions":[{"question":"Q?","answer":"long answer","key_points":["a","b","c"]}]}`, models.TenMark, 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			qs, err := ParseQuestions(tc.raw, tc.qt, tc.want)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected ErrInvalid, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(qs) != tc.want {
				t.Fatalf("expected %d questions, got %d", tc.want, len(qs))
			}
		})
	}
}
