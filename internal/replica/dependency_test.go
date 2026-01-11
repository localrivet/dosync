/*
Copyright © 2025 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"testing"
)

func TestGetServiceDependencies(t *testing.T) {
	tests := []struct {
		name        string
		compose     string
		serviceName string
		wantDeps    []string
		wantErr     bool
	}{
		{
			name: "service with health check dependency",
			compose: `
services:
  postgres:
    image: postgres:16
  app:
    image: myapp:latest
    depends_on:
      postgres:
        condition: service_healthy
`,
			serviceName: "app",
			wantDeps:    []string{"postgres"},
			wantErr:     false,
		},
		{
			name: "service with multiple dependencies",
			compose: `
services:
  postgres:
    image: postgres:16
  redis:
    image: redis:7
  app:
    image: myapp:latest
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
`,
			serviceName: "app",
			wantDeps:    []string{"postgres", "redis"},
			wantErr:     false,
		},
		{
			name: "service with no health check condition",
			compose: `
services:
  postgres:
    image: postgres:16
  app:
    image: myapp:latest
    depends_on:
      postgres:
        condition: service_started
`,
			serviceName: "app",
			wantDeps:    []string{},
			wantErr:     false,
		},
		{
			name: "service with no dependencies",
			compose: `
services:
  app:
    image: myapp:latest
`,
			serviceName: "app",
			wantDeps:    []string{},
			wantErr:     false,
		},
		{
			name: "service not found",
			compose: `
services:
  app:
    image: myapp:latest
`,
			serviceName: "nonexistent",
			wantDeps:    nil,
			wantErr:     true,
		},
		{
			name: "mixed dependencies - only health check ones returned",
			compose: `
services:
  postgres:
    image: postgres:16
  redis:
    image: redis:7
  app:
    image: myapp:latest
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
`,
			serviceName: "app",
			wantDeps:    []string{"postgres"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := GetServiceDependencies([]byte(tt.compose), tt.serviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check that all expected deps are present (order doesn't matter)
			if len(deps) != len(tt.wantDeps) {
				t.Errorf("GetServiceDependencies() got %v deps, want %v", len(deps), len(tt.wantDeps))
				return
			}

			depsMap := make(map[string]bool)
			for _, d := range deps {
				depsMap[d] = true
			}
			for _, want := range tt.wantDeps {
				if !depsMap[want] {
					t.Errorf("GetServiceDependencies() missing dependency %s", want)
				}
			}
		})
	}
}

func TestDefaultDependencyHealthConfig(t *testing.T) {
	config := DefaultDependencyHealthConfig()

	if config.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if config.PollInterval <= 0 {
		t.Error("PollInterval should be positive")
	}
	if config.Timeout < config.PollInterval {
		t.Error("Timeout should be greater than PollInterval")
	}
}
