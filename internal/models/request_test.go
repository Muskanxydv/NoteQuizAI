package models

import "testing"

func TestGenerateRequestValidate(t *testing.T) {
	ok := GenerateRequest{Notes: "notes", Questions: []QuestionSpec{{Type: OneMark, Count: 2}, {Type: TenMark, Count: 1}}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	bad := map[string]GenerateRequest{
		"empty notes":    {Notes: " ", Questions: ok.Questions},
		"no questions":   {Notes: "notes"},
		"unknown type":   {Notes: "notes", Questions: []QuestionSpec{{Type: "5-mark", Count: 1}}},
		"zero count":     {Notes: "notes", Questions: []QuestionSpec{{Type: OneMark, Count: 0}}},
		"too many":       {Notes: "notes", Questions: []QuestionSpec{{Type: OneMark, Count: MaxCountPerType + 1}}},
		"duplicate type": {Notes: "notes", Questions: []QuestionSpec{{Type: OneMark, Count: 1}, {Type: OneMark, Count: 2}}},
	}
	for name, req := range bad {
		if err := req.Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
