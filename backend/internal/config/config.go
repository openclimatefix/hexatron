package config

import (
	"strings"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

// Load reads environment variables and returns a populated Config.
// Missing variables fall back to safe defaults.
func Load() *configstructs.Config {
	port := Env("PORT", ":8080")
	if !strings.HasPrefix(port, ":") && !strings.Contains(port, ":") {
		port = ":" + port
	}

	return &configstructs.Config{
		Addr:               port,
		AirflowBaseURL:     Env(constants.AirflowBaseURLEnv, constants.AirflowDefaultURL),
		AirflowCookie:      Env(constants.AirflowCookieEnv, ""),
		ServicesConfigPath: Env("SERVICES_CONFIG_PATH", constants.ServicesConfigPath),
	}
}
