// Package config defines the application configuration structure.
package config

// Config holds all runtime configuration for the Hexatron backend.
//
//	Addr               – address the HTTP server listens on (e.g. ":8080")
//	AirflowBaseURL     – Airflow REST API base URL
//	AirflowCookie      – session cookie for local-development authentication
//	ServicesConfigPath – path to data/services.yaml
type Config struct {
	Addr               string
	AirflowBaseURL     string
	AirflowCookie      string
	ServicesConfigPath string
}
