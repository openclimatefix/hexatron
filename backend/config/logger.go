package config

import "log/slog"

// NewLogger returns the default structured logger.
// Swap for a custom JSON handler before deploying to production.
func NewLogger() *slog.Logger {
	return slog.Default()
}
