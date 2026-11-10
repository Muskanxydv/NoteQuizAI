// Command probe sends ONE request to check that your API key and model IDs work.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"notequizai/internal/config"
	"notequizai/internal/llm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := llm.NewPool(llm.PoolConfig{
		APIKey:         cfg.GeminiAPIKey,
		Models:         cfg.GeminiModels,
		RPM:            cfg.RPMPerModel,
		RPD:            cfg.RPDPerModel,
		RequestTimeout: cfg.RequestTimeout,
		MaxRetries:     cfg.MaxRetries,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	out, err := pool.Generate(ctx, `Reply with exactly this JSON and nothing else: {"ok": true}`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Gemini replied:", out)
}
