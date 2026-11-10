package parser

import (
	"strings"
	"testing"
)

var filler = strings.Repeat("word ", 50) // 250 characters

func TestSplitByHeadings(t *testing.T) {
	notes := "# Photosynthesis\n" + filler + "\n\n# Respiration\n" + filler
	topics := Split(notes)
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}
	if topics[0].Title != "Photosynthesis" || topics[1].Title != "Respiration" {
		t.Fatalf("unexpected titles: %q, %q", topics[0].Title, topics[1].Title)
	}
}

func TestSplitByParagraphs(t *testing.T) {
	notes := filler + "\n\n" + filler + "\n\n" + filler
	topics := Split(notes)
	if len(topics) != 3 {
		t.Fatalf("expected 3 topics, got %d", len(topics))
	}
	if topics[0].Title != "Topic 1" {
		t.Fatalf("expected fallback title, got %q", topics[0].Title)
	}
}

func TestSmallChunksAreMerged(t *testing.T) {
	topics := Split("short one\n\nshort two\n\nshort three")
	if len(topics) != 1 {
		t.Fatalf("expected 1 merged topic, got %d", len(topics))
	}
}

func TestTopicCountIsCapped(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("# Section\n" + filler + "\n\n")
	}
	if n := len(Split(sb.String())); n > MaxTopics {
		t.Fatalf("expected at most %d topics, got %d", MaxTopics, n)
	}
}

func TestEmptyNotes(t *testing.T) {
	if topics := Split("  \n "); topics != nil {
		t.Fatalf("expected nil, got %v", topics)
	}
}

func TestWindowsLineEndings(t *testing.T) {
	notes := "# A\r\n" + filler + "\r\n\r\n# B\r\n" + filler
	if n := len(Split(notes)); n != 2 {
		t.Fatalf("expected 2 topics, got %d", n)
	}
}
