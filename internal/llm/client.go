package llm

import "context"

// Client is the only thing the rest of the app knows about the AI provider.
// Implementations (e.g. Gemini) live in this package.
type Client interface {
	// Generate sends a prompt and returns the model's raw text response
	// (expected to be JSON). It must respect ctx cancellation and deadlines.
	Generate(ctx context.Context, prompt string) (string, error)
}
