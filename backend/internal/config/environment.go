package config

import "os"

// Env returns the value of the environment variable named by key.
// If the variable is not set or is empty, fallback is returned.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
