// Package services implements business logic and configuration registry lookups.
package services

import (
	"fmt"
	"os"

	"github.com/openclimatefix/hexatron/backend/internal/models"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
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

	if err := validate(cfg.Services); err != nil {
		return nil, fmt.Errorf("invalid services config %s: %w", configPath, err)
	}

	return &ServiceRegistry{services: cfg.Services}, nil
}

// validate rejects duplicate ids, dangling depends_on edges and malformed globs.
func validate(svcs []models.Service) error {
	ids := make(map[string]struct{}, len(svcs))

	for _, svc := range svcs {
		if svc.ID == "" {
			return fmt.Errorf("service %q has no id", svc.Name)
		}
		if _, duplicate := ids[svc.ID]; duplicate {
			return fmt.Errorf("duplicate service id %q", svc.ID)
		}
		ids[svc.ID] = struct{}{}

		for _, pattern := range svc.DAGPatterns {
			if err := models.ValidateDAGPattern(pattern); err != nil {
				return fmt.Errorf("service %q: dag pattern %q: %w", svc.ID, pattern, err)
			}
		}
	}

	// Checked in a second pass so a service may depend on one defined below it.
	for _, svc := range svcs {
		for _, dep := range svc.DependsOn {
			if _, known := ids[dep]; !known {
				return fmt.Errorf("service %q depends_on unknown service %q", svc.ID, dep)
			}
		}
	}
	return nil
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
