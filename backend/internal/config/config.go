package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr              string
	DatabaseURL       string
	DefaultModel      string
	ContextCharBudget int
	ReplyCharReserve  int
	FrontendOrigin    string
	StaticDir         string
}

func Load() Config {
	loadDotEnv(".env")
	loadDotEnv("../.env")
	return Config{
		Addr:              getEnv("ADDR", ":8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/book_world?sslmode=disable"),
		DefaultModel:      getEnv("DEFAULT_MODEL", "gpt-4o-mini"),
		ContextCharBudget: getEnvInt("CONTEXT_CHAR_BUDGET", 48000),
		ReplyCharReserve:  getEnvInt("REPLY_CHAR_RESERVE", 6000),
		FrontendOrigin:    getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		StaticDir:         getEnv("STATIC_DIR", ""),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
