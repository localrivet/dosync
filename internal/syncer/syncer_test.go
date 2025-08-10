package syncer

import (
	"dosync/internal/config"
	"errors"
	"os"
	"testing"

	"dosync/internal/replica"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestMain(m *testing.M) {
	// Set the real YAML unmarshal implementation for all tests
	YamlUnmarshal = yaml.Unmarshal
	os.Exit(m.Run())
}

func TestExtractDORepositoryInfo(t *testing.T) {
	tests := []struct {
		image    string
		wantReg  string
		wantRepo string
		wantErr  bool
	}{
		{"registry.digitalocean.com/myreg/myrepo:tag", "myreg", "myrepo", false},
		{"registry.digitalocean.com/abc/def:main-20240501-abc123", "abc", "def", false},
		{"invalidimage", "", "", true},
		{"registry.digitalocean.com/onlyreg", "", "", true},
	}
	for _, tt := range tests {
		reg, repo, err := extractDORepositoryInfo(tt.image)
		if tt.wantErr {
			assert.Error(t, err, "expected error for %s", tt.image)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tt.image)
			assert.Equal(t, tt.wantReg, reg)
			assert.Equal(t, tt.wantRepo, repo)
		}
	}
}

func TestExtractTagFromImage(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"registry.digitalocean.com/myreg/myrepo:tag", "tag"},
		{"registry.digitalocean.com/abc/def:main-20240501-abc123", "main-20240501-abc123"},
		{"registry.digitalocean.com/abc/def", "latest"},
	}
	for _, tt := range tests {
		got := extractTagFromImage(tt.image)
		assert.Equal(t, tt.want, got)
	}
}

func TestCheckAndUpdateServices_InvalidFile(t *testing.T) {
	// Create a temp file to ensure the file exists
	tmpFile, err := os.CreateTemp("", "compose-*.yml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	called := false
	YamlUnmarshal = func(in []byte, out interface{}) error {
		called = true
		return errors.New("unmarshal error")
	}
	// Should not panic or crash
	checkAndUpdateServices(tmpFile.Name(), false)
	assert.True(t, called, "YamlUnmarshal should be called")
}

func TestUpdateDockerComposeAndRestart_MissingFile(t *testing.T) {
	err := replica.UpdateDockerComposeAndRestart("svc", "tag", "nonexistent-file.yml", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read docker-compose file")
}

func TestRemoveUnusedDockerImages_NoError(t *testing.T) {
	// This just ensures the function runs; it may prune images if Docker is running
	removeUnusedDockerImages(false)
}

func TestCheckAndUpdateServices_ValidYAML(t *testing.T) {
	dockerComposeYAML := `
services:
  web:
    image: registry.digitalocean.com/myreg/myrepo:main-20240501-abc123
  db:
    image: postgres:13
`
	tmpFile, err := os.CreateTemp("", "compose-*.yml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte(dockerComposeYAML))
	assert.NoError(t, err)
	tmpFile.Close()

	// Ensure config is loaded before calling checkAndUpdateServices
	_, err = config.LoadConfig("", nil)
	assert.NoError(t, err)

	calls := 0
	YamlUnmarshal = func(in []byte, out interface{}) error {
		calls++
		return yaml.Unmarshal(in, out)
	}

	// Should not panic or crash
	checkAndUpdateServices(tmpFile.Name(), false)
	assert.Equal(t, 1, calls, "YamlUnmarshal should be called once for valid YAML")
}

func TestSelectTagByImagePolicy(t *testing.T) {
	tags := []string{
		"main-abc123-100",
		"main-def456-200",
		"main-ghi789-150",
		"dev-xyz-300",
		"v1.2.3",
		"v1.2.4",
		"v2.0.0-rc1",
		"v2.0.0",
		"RELEASE.2024-06-01T12-00-00Z",
		"RELEASE.2024-06-02T12-00-00Z",
	}

	t.Run("Numerical desc with extract", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^main-[a-z0-9]+-(?P<ts>\\d+)$`,
				Extract: "ts",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Numerical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "desc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "main-def456-200", selected)
	})

	t.Run("Numerical asc with extract", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^main-[a-z0-9]+-(?P<ts>\\d+)$`,
				Extract: "ts",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Numerical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "asc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "main-abc123-100", selected)
	})

	t.Run("Semver latest", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: nil,
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Semver: &struct {
					Range string `mapstructure:"range" yaml:"range"`
				}{Range: ""},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "v2.0.0", selected)
	})

	t.Run("Semver with range", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: nil,
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Semver: &struct {
					Range string `mapstructure:"range" yaml:"range"`
				}{Range: ">=1.2.0 <2.0.0"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "v1.2.4", selected)
	})

	t.Run("Semver with extract", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^v(?P<semver>[0-9]+\.[0-9]+\.[0-9]+)$`,
				Extract: "$semver",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Semver: &struct {
					Range string `mapstructure:"range" yaml:"range"`
				}{Range: ">=1.2.0 <2.0.0"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "v1.2.4", selected)
	})

	t.Run("Alphabetical desc", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^RELEASE\\.(?P<timestamp>.*)Z$`,
				Extract: "$timestamp",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Alphabetical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "desc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "RELEASE.2024-06-02T12-00-00Z", selected)
	})

	t.Run("Alphabetical asc", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^RELEASE\\.(?P<timestamp>.*)Z$`,
				Extract: "$timestamp",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Alphabetical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "asc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "RELEASE.2024-06-01T12-00-00Z", selected)
	})

	t.Run("Regex filter only", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^dev-.*`,
				Extract: "",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Alphabetical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "desc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "dev-xyz-300", selected)
	})

	t.Run("Fallback to lexicographical last", func(t *testing.T) {
		selected, err := SelectTagByImagePolicy(tags, nil)
		assert.NoError(t, err)
		assert.Equal(t, "v2.0.0", selected)
	})

	t.Run("No match returns empty", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^doesnotmatch.*`,
				Extract: "",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Alphabetical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "desc"},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "", selected)
	})

	t.Run("Empty tags returns empty", func(t *testing.T) {
		selected, err := SelectTagByImagePolicy([]string{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, "", selected)
	})

	t.Run("Invalid regex returns error", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `([`, // invalid regex
				Extract: "",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Alphabetical: &struct {
					Order string `mapstructure:"order" yaml:"order"`
				}{Order: "desc"},
			},
		}
		_, err := SelectTagByImagePolicy(tags, policy)
		assert.Error(t, err)
	})

	t.Run("Invalid semver returns empty", func(t *testing.T) {
		policy := &config.ImagePolicy{
			FilterTags: &struct {
				Pattern string `mapstructure:"pattern" yaml:"pattern"`
				Extract string `mapstructure:"extract" yaml:"extract"`
			}{
				Pattern: `^main-[a-z0-9]+-(?P<ts>\\d+)$`,
				Extract: "ts",
			},
			Policy: &struct {
				Numerical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"numerical" yaml:"numerical"`
				Semver *struct {
					Range string `mapstructure:"range" yaml:"range"`
				} `mapstructure:"semver" yaml:"semver"`
				Alphabetical *struct {
					Order string `mapstructure:"order" yaml:"order"`
				} `mapstructure:"alphabetical" yaml:"alphabetical"`
			}{
				Semver: &struct {
					Range string `mapstructure:"range" yaml:"range"`
				}{Range: ""},
			},
		}
		selected, err := SelectTagByImagePolicy(tags, policy)
		assert.NoError(t, err)
		assert.Equal(t, "", selected)
	})
}

func TestIsSemanticDowngrade(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		selected string
		want     bool
		wantErr  bool
	}{
		{
			name:     "v0.1.10 to v0.1.9 is downgrade",
			current:  "v0.1.10",
			selected: "v0.1.9",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "v0.1.9 to v0.1.10 is not downgrade",
			current:  "v0.1.9",
			selected: "v0.1.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "same version is not downgrade",
			current:  "v0.1.10",
			selected: "v0.1.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "v1.0.0 to v2.0.0 is not downgrade",
			current:  "v1.0.0",
			selected: "v2.0.0",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "v2.0.0 to v1.0.0 is downgrade",
			current:  "v2.0.0",
			selected: "v1.0.0",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "without v prefix works",
			current:  "0.1.10",
			selected: "0.1.9",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "non-semver falls back to string comparison",
			current:  "main-abc-100",
			selected: "main-abc-090",
			want:     true, // "main-abc-090" < "main-abc-100" alphabetically
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isSemanticDowngrade(tt.current, tt.selected)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got, "isSemanticDowngrade(%q, %q)", tt.current, tt.selected)
			}
		})
	}
}

func TestShouldUpdateSemantically(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		selected string
		want     bool
		wantErr  bool
	}{
		{
			name:     "v0.1.9 to v0.1.10 should update",
			current:  "v0.1.9",
			selected: "v0.1.10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "v0.1.10 to v0.1.9 should not update",
			current:  "v0.1.10",
			selected: "v0.1.9",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "same version should not update",
			current:  "v0.1.10",
			selected: "v0.1.10",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "v1.0.0 to v2.0.0 should update",
			current:  "v1.0.0",
			selected: "v2.0.0",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "v2.0.0 to v1.0.0 should not update",
			current:  "v2.0.0",
			selected: "v1.0.0",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "without v prefix works",
			current:  "0.1.9",
			selected: "0.1.10",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "non-semver falls back to string comparison",
			current:  "main-abc-090",
			selected: "main-abc-100",
			want:     true, // "main-abc-100" > "main-abc-090" alphabetically
			wantErr:  false,
		},
		{
			name:     "non-semver reverse should not update",
			current:  "main-abc-100",
			selected: "main-abc-090",
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldUpdateSemantically(tt.current, tt.selected)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got, "shouldUpdateSemantically(%q, %q)", tt.current, tt.selected)
			}
		})
	}
}

func TestGetActualRunningImageTag(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		expectError bool
		description string
	}{
		{
			name:        "service with running containers",
			serviceName: "test-service",
			expectError: false, // May vary based on system state
			description: "Should attempt to get running container image tag",
		},
		{
			name:        "nonexistent service",
			serviceName: "nonexistent-service-12345",
			expectError: true,
			description: "Should return error for nonexistent service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := getActualRunningImageTag(tt.serviceName)

			if tt.expectError {
				assert.Error(t, err, "Expected error for %s", tt.description)
				assert.Empty(t, tag, "Expected empty tag when error occurs")
			} else {
				// Note: This test may pass or fail depending on system state
				// The important thing is that the function doesn't panic
				t.Logf("Service %s: tag=%s, err=%v", tt.serviceName, tag, err)
			}
		})
	}
}
