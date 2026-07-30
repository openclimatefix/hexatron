// Package services implements business logic and configuration registry lookups.
package services

import (
	"fmt"
	"os"

	"github.com/openclimatefix/hexatron/backend/models"
	configstructs "github.com/openclimatefix/hexatron/backend/structures/config"
	"gopkg.in/yaml.v3"
)

// ServiceRegistry loads and stores business services from services.yaml.
type ServiceRegistry struct {
	services []models.Service
}

// NewServiceRegistry parses the services.yaml file at the given configPath.
func NewServiceRegistry(configPath string) (*ServiceRegistry, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading services config from %s: %w", configPath, err)
	}

	var cfg configstructs.ServicesYAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing services yaml: %w", err)
	}

	return &ServiceRegistry{services: cfg.Services}, nil
}

// All returns all configured services loaded from services.yaml.
func (r *ServiceRegistry) All() []models.Service {
	return r.services
}

// ByID returns a configured service by its ID, and boolean indicating if found.
func (r *ServiceRegistry) ByID(id string) (models.Service, bool) {
	for _, svc := range r.services {
		if svc.ID == id {
			return svc, true
		}
	}
	return models.Service{}, false
}
