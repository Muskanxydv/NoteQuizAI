package generator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"notequizai/internal/models"
)

func benchGenerate(b *testing.B, workers int) {
	topics := make([]models.Topic, 9)
	for i := range topics {
		topics[i] = models.Topic{Title: fmt.Sprintf("T%d", i), Content: "x"}
	}
	g := New(&fakeClient{delay: 20 * time.Millisecond, respond: alwaysValid}, workers, 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := g.Generate(context.Background(), topics, oneMark3); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSequential(b *testing.B) { benchGenerate(b, 1) }
func BenchmarkConcurrent(b *testing.B) { benchGenerate(b, 9) }
