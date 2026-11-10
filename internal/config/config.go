package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultModels = "gemini-3.5-flash-lite,gemini-3.1-flash-lite"

type Config struct {
	Port            string
	GeminiAPIKey    string
	GeminiModels    []string
	RPMPerModel     int
	RPDPerModel     int
	MaxWorkers      int
	MaxRetries      int
	RequestTimeout  time.Duration
	GenerateTimeout time.Duration
	Host            string
}

func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Host:         getEnv("HOST", "127.0.0.1"),
		Port:         getEnv("PORT", "8080"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		GeminiModels: splitList(getEnv("GEMINI_MODELS", defaultModels)),
	}
	if cfg.GeminiAPIKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not set (put it in .env or your environment)")
	}
	if len(cfg.GeminiModels) == 0 {
		return nil, errors.New("GEMINI_MODELS must list at least one model")
	}

	var err error
	if cfg.RPMPerModel, err = getInt("GEMINI_RPM", 14, 1); err != nil {
		return nil, err
	}
	if cfg.RPDPerModel, err = getInt("GEMINI_RPD", 480, 1); err != nil {
		return nil, err
	}
	if cfg.MaxWorkers, err = getInt("MAX_WORKERS", 6, 1); err != nil {
		return nil, err
	}
	if cfg.MaxRetries, err = getInt("MAX_RETRIES", 3, 0); err != nil {
		return nil, err
	}
	cfg.RequestTimeout, err = time.ParseDuration(getEnv("REQUEST_TIMEOUT", "60s"))
	if err != nil || cfg.RequestTimeout <= 0 {
		return nil, fmt.Errorf("REQUEST_TIMEOUT must be a positive duration like 60s")
	}
	cfg.GenerateTimeout, err = time.ParseDuration(getEnv("GENERATE_TIMEOUT", "120s"))
	if err != nil || cfg.GenerateTimeout <= 0 {
		return nil, fmt.Errorf("GENERATE_TIMEOUT must be a positive duration like 120s")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getInt(key string, def, min int) (int, error) {
	v := getEnv(key, "")
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min {
		return 0, fmt.Errorf("%s must be an integer >= %d, got %q", key, min, v)
	}
	return n, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
