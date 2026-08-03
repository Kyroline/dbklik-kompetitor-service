// Package config provides small, reusable helpers for reading environment
// variables with defaults. internal/config builds the app-wide Config
// struct on top of these helpers.
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads a .env file if present. Missing files are not an error,
// since production environments typically inject real env vars instead.
func LoadDotEnv(path string) {
	_ = godotenv.Load(path)
}

func GetString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func GetInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func GetBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
