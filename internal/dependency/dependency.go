package dependency

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// DependencyManager defines the interface for managing service dependencies
// in a Docker Compose environment.
type DependencyManager interface {
	BuildDependencyGraph(composeFile string) (*DependencyGraph, error)
	GetUpdateOrder(services []string) ([]string, error)
	GetDependents(service string) ([]string, error)
	ShouldUpdateDependents(service string, updateType string) bool
	GetServiceDependencies(service string) ([]string, error)
}

// DependencyGraph represents the directed graph of service dependencies.
type DependencyGraph struct {
	Services map[string][]string // Map of service to its dependencies
}

// dependencyManager is a concrete implementation of DependencyManager
// (unexported, use NewDependencyManager to construct)
type dependencyManager struct {
	graph *DependencyGraph
}

// NewDependencyManager creates a new DependencyManager from a compose file
func NewDependencyManager(composeFile string) (DependencyManager, error) {
	mgr := &dependencyManager{
		graph: &DependencyGraph{Services: make(map[string][]string)},
	}
	_, err := mgr.BuildDependencyGraph(composeFile)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func (dm *dependencyManager) BuildDependencyGraph(composeFile string) (*DependencyGraph, error) {
	file, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose struct {
		Services map[string]struct {
			DependsOn interface{} `yaml:"depends_on"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(file, &compose); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	graph := &DependencyGraph{Services: make(map[string][]string)}
	for svc, def := range compose.Services {
		var deps []string
		switch v := def.DependsOn.(type) {
		case []interface{}:
			for _, dep := range v {
				if depStr, ok := dep.(string); ok {
					deps = append(deps, depStr)
				}
			}
		case map[string]interface{}:
			for dep := range v {
				deps = append(deps, dep)
			}
		}
		graph.Services[svc] = deps
	}
	dm.graph = graph
	return graph, nil
}

func (dm *dependencyManager) GetUpdateOrder(services []string) ([]string, error) {
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	added := make(map[string]bool)
	var result []string
	var visit func(string) error

	visit = func(n string) error {
		if tempMark[n] {
			return fmt.Errorf("circular dependency detected at service: %s", n)
		}
		if !visited[n] {
			tempMark[n] = true
			// Sort dependencies for deterministic output
			deps := append([]string{}, dm.graph.Services[n]...)
			sort.Strings(deps)
			for _, dep := range deps {
				if err := visit(dep); err != nil {
					return err
				}
			}
			tempMark[n] = false
			visited[n] = true
			if !added[n] {
				result = append(result, n)
				added[n] = true
			}
		}
		return nil
	}

	// Sort input services for deterministic output
	input := append([]string{}, services...)
	sort.Strings(input)
	for _, svc := range input {
		if err := visit(svc); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (dm *dependencyManager) GetDependents(service string) ([]string, error) {
	// Find all services that (directly or indirectly) depend on the given service
	dependents := make(map[string]bool)
	for svc := range dm.graph.Services {
		if svc == service {
			continue
		}
		if dependsOn(dm.graph, svc, service, make(map[string]bool)) {
			dependents[svc] = true
		}
	}
	var result []string
	for dep := range dependents {
		result = append(result, dep)
	}
	return result, nil
}

// Helper: recursively check if start depends on target
func dependsOn(graph *DependencyGraph, start, target string, visited map[string]bool) bool {
	if visited[start] {
		return false
	}
	visited[start] = true
	for _, dep := range graph.Services[start] {
		if dep == target || dependsOn(graph, dep, target, visited) {
			return true
		}
	}
	return false
}

func (dm *dependencyManager) ShouldUpdateDependents(service string, updateType string) bool {
	// For now, return true if any dependents exist
	deps, err := dm.GetDependents(service)
	return err == nil && len(deps) > 0
}

func (dm *dependencyManager) GetServiceDependencies(service string) ([]string, error) {
	deps, ok := dm.graph.Services[service]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", service)
	}
	return deps, nil
}
