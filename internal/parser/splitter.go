// Package parser turns unstructured study notes into topics that can be
// processed concurrently.
package parser

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"notequizai/internal/models"
)

const (
	// MinTopicChars is the smallest chunk worth its own LLM request; smaller
	// chunks are merged with their neighbours.
	MinTopicChars = 200
	// MaxTopics caps the fan-out so one huge paste cannot burn the daily quota.
	MaxTopics = 6
)

var blankLine = regexp.MustCompile(`\n[ \t]*\n`)

// Split breaks notes into topics. It splits on markdown headings (# Title)
// when present, otherwise on blank lines, then merges tiny chunks and caps
// the total number of topics.
func Split(notes string) []models.Topic {
	text := strings.TrimSpace(strings.ReplaceAll(notes, "\r\n", "\n"))
	if text == "" {
		return nil
	}

	blocks := splitByHeadings(text)
	if len(blocks) <= 1 {
		blocks = nil
		for _, p := range blankLine.Split(text, -1) {
			if p = strings.TrimSpace(p); p != "" {
				blocks = append(blocks, p)
			}
		}
	}
	blocks = mergeSmall(blocks)
	blocks = limitCount(blocks, MaxTopics)

	topics := make([]models.Topic, 0, len(blocks))
	for i, b := range blocks {
		topics = append(topics, models.Topic{Title: deriveTitle(b, i), Content: b})
	}
	return topics
}

func isHeading(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "#") && strings.HasPrefix(strings.TrimLeft(t, "#"), " ")
}

func splitByHeadings(text string) []string {
	var blocks, cur []string
	flush := func() {
		if s := strings.TrimSpace(strings.Join(cur, "\n")); s != "" {
			blocks = append(blocks, s)
		}
		cur = nil
	}
	for _, line := range strings.Split(text, "\n") {
		if isHeading(line) {
			flush()
		}
		cur = append(cur, line)
	}
	flush()
	return blocks
}

// mergeSmall accumulates blocks until each chunk reaches MinTopicChars.
func mergeSmall(blocks []string) []string {
	var out []string
	cur := ""
	for _, b := range blocks {
		if cur == "" {
			cur = b
		} else {
			cur += "\n\n" + b
		}
		if len(cur) >= MinTopicChars {
			out = append(out, cur)
			cur = ""
		}
	}
	if cur != "" {
		if len(out) > 0 {
			out[len(out)-1] += "\n\n" + cur
		} else {
			out = append(out, cur)
		}
	}
	return out
}

// limitCount groups neighbouring blocks so at most max remain.
func limitCount(blocks []string, max int) []string {
	if len(blocks) <= max {
		return blocks
	}
	per := (len(blocks) + max - 1) / max
	var out []string
	for i := 0; i < len(blocks); i += per {
		end := i + per
		if end > len(blocks) {
			end = len(blocks)
		}
		out = append(out, strings.Join(blocks[i:end], "\n\n"))
	}
	return out
}

// deriveTitle uses the first line when it looks like a heading, else "Topic N".
func deriveTitle(body string, idx int) string {
	first, _, _ := strings.Cut(body, "\n")
	first = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(first), "#"))
	first = strings.TrimRight(first, ": ")
	if first != "" && utf8.RuneCountInString(first) <= 80 && !strings.HasSuffix(first, ".") {
		return first
	}
	return fmt.Sprintf("Topic %d", idx+1)
}
