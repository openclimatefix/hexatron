// Package models holds the shared domain types used across all layers.
package models

// Service represents a business service defined in data/services.yaml.
type Service struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Category string   `yaml:"category"`
	DAGIDs   []string `yaml:"dags"`
}
