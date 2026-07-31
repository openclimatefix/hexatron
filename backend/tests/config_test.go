package tests

import (
	"os"
	"testing"

	"github.com/openclimatefix/hexatron/backend/config"
	"github.com/openclimatefix/hexatron/backend/constants"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv(constants.AirflowBaseURLEnv)
	os.Unsetenv(constants.AirflowCookieEnv)

	cfg := config.Load()

	if cfg.Addr != ":8080" {
		t.Errorf("expected default Addr :8080, got %s", cfg.Addr)
	}
	if cfg.AirflowBaseURL != constants.AirflowDefaultURL {
		t.Errorf("expected default AirflowBaseURL %s, got %s", constants.AirflowDefaultURL, cfg.AirflowBaseURL)
	}
	if cfg.AirflowCookie != "" {
		t.Errorf("expected empty default AirflowCookie, got %s", cfg.AirflowCookie)
	}
}

func TestLoadCustomEnv(t *testing.T) {
	os.Setenv("PORT", ":9090")
	os.Setenv(constants.AirflowBaseURLEnv, "http://airflow.example.com")
	os.Setenv(constants.AirflowCookieEnv, "session=secret")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv(constants.AirflowBaseURLEnv)
		os.Unsetenv(constants.AirflowCookieEnv)
	}()

	cfg := config.Load()

	if cfg.Addr != ":9090" {
		t.Errorf("expected Addr :9090, got %s", cfg.Addr)
	}
	if cfg.AirflowBaseURL != "http://airflow.example.com" {
		t.Errorf("expected AirflowBaseURL http://airflow.example.com, got %s", cfg.AirflowBaseURL)
	}
	if cfg.AirflowCookie != "session=secret" {
		t.Errorf("expected AirflowCookie session=secret, got %s", cfg.AirflowCookie)
	}
}
