// Package config loads application configuration from environment variables.
package config

import (
	"github.com/openclimatefix/hexatron/backend/constants"
	configstructs "github.com/openclimatefix/hexatron/backend/structures/config"
)

// Load reads environment variables and returns a populated Config.
// Missing variables fall back to safe defaults.
func Load() *configstructs.Config {
	return &configstructs.Config{
		Addr:           Env("PORT", ":8080"),
		AirflowBaseURL: Env(constants.AirflowBaseURLEnv, constants.AirflowDefaultURL),
		AirflowCookie:  Env(constants.AirflowCookieEnv, ""),
	}
}
