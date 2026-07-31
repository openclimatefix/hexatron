// Package config defines YAML configuration structures.
package config

import "github.com/openclimatefix/hexatron/backend/internal/models"

// ServicesYAMLConfig mirrors the top-level structure of data/services.yaml.
type ServicesYAMLConfig struct {
	Services []models.Service `yaml:"services"`
}
