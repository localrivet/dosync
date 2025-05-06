package config

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	origEnv = map[string]string{
		"CHECK_INTERVAL": os.Getenv("CHECK_INTERVAL"),
		"VERBOSE":        os.Getenv("VERBOSE"),
	}
)

func resetConfigTestEnv() {
	os.Unsetenv("CHECK_INTERVAL")
	os.Unsetenv("VERBOSE")
	cfg = nil
	cfgOnce = sync.Once{}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	resetConfigTestEnv()
	os.Setenv("CHECK_INTERVAL", "5m")
	os.Setenv("VERBOSE", "true")
	defer resetConfigTestEnv()

	cfg, err := LoadConfig("", nil)
	assert.NoError(t, err)
	assert.Equal(t, "5m", cfg.CheckInterval)
	assert.Equal(t, true, cfg.Verbose)
}

func TestLoadConfig_Defaults(t *testing.T) {
	resetConfigTestEnv()
	cfg, err := LoadConfig("", nil)
	assert.NoError(t, err)
	assert.Equal(t, "1m", cfg.CheckInterval)
	assert.Equal(t, false, cfg.Verbose)
}

func TestLoadConfig_FlagsOverride(t *testing.T) {
	resetConfigTestEnv()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("CHECK_INTERVAL", "10m", "")
	flags.Bool("VERBOSE", true, "")
	flags.Parse([]string{"--CHECK_INTERVAL=10m", "--VERBOSE=true"})

	cfg, err := LoadConfig("", flags)
	assert.NoError(t, err)
	assert.Equal(t, "10m", cfg.CheckInterval)
	assert.Equal(t, true, cfg.Verbose)
}

func TestLoadConfig_RegistrySection(t *testing.T) {
	resetConfigTestEnv()
	defer resetConfigTestEnv()

	yaml := `
registry:
  dockerhub:
    username: testuser
    password: testpass
  gcr:
    credentials_file: /tmp/gcp.json
  ghcr:
    token: ghcrtoken
  acr:
    tenant_id: tid
    client_id: cid
    client_secret: csecret
    registry: acr.azurecr.io
  quay:
    token: quaytoken
  harbor:
    url: https://harbor.example.com
    username: huser
    password: hpass
  docr:
    token: doctoken
  ecr:
    aws_access_key_id: ecrkey
    aws_secret_access_key: ecrsecret
    region: us-east-1
    registry: 123456789012.dkr.ecr.us-east-1.amazonaws.com
  custom:
    url: https://custom.example.com
    username: cuser
    password: cpass
`
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(yaml))
	assert.NoError(t, err)
	var c Config
	err = v.Unmarshal(&c)
	assert.NoError(t, err)
	if assert.NotNil(t, c.Registry) {
		assert.Equal(t, "testuser", c.Registry.DockerHub.Username)
		assert.Equal(t, "/tmp/gcp.json", c.Registry.GCR.CredentialsFile)
		assert.Equal(t, "ghcrtoken", c.Registry.GHCR.Token)
		assert.Equal(t, "tid", c.Registry.ACR.TenantID)
		assert.Equal(t, "quaytoken", c.Registry.Quay.Token)
		assert.Equal(t, "https://harbor.example.com", c.Registry.Harbor.URL)
		assert.Equal(t, "doctoken", c.Registry.DOCR.Token)
		assert.Equal(t, "ecrkey", c.Registry.ECR.AWSAccessKeyID)
		assert.Equal(t, "https://custom.example.com", c.Registry.Custom.URL)
	}
}

func TestLoadConfig_RegistrySection_Partial(t *testing.T) {
	resetConfigTestEnv()
	defer resetConfigTestEnv()
	yaml := `
registry:
  dockerhub:
    username: onlyuser
`
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(yaml))
	assert.NoError(t, err)
	var c Config
	err = v.Unmarshal(&c)
	assert.NoError(t, err)
	if assert.NotNil(t, c.Registry) {
		assert.Equal(t, "onlyuser", c.Registry.DockerHub.Username)
		assert.Nil(t, c.Registry.GCR)
	}
}

func TestLoadConfig_RegistrySection_EnvExpansion(t *testing.T) {
	resetConfigTestEnv()
	defer resetConfigTestEnv()
	os.Setenv("DOCKERHUB_PASSWORD", "envpass")
	yaml := `
registry:
  dockerhub:
    username: envuser
    password: ${DOCKERHUB_PASSWORD}
`
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(yaml))
	assert.NoError(t, err)
	var c Config
	err = v.Unmarshal(&c)
	assert.NoError(t, err)
	if assert.NotNil(t, c.Registry) {
		// Viper does not expand env vars by default, so simulate expansion
		pass := os.ExpandEnv(c.Registry.DockerHub.Password)
		assert.Equal(t, "envpass", pass)
	}
}

func TestValidateImagePolicy(t *testing.T) {
	valid := &ImagePolicy{
		FilterTags: &struct {
			Pattern string `mapstructure:"pattern" yaml:"pattern"`
			Extract string `mapstructure:"extract" yaml:"extract"`
		}{
			Pattern: `^main-[a-z0-9]+-(?P<ts>\\d+)$`,
			Extract: "$ts",
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
			Semver: &struct {
				Range string `mapstructure:"range" yaml:"range"`
			}{Range: ">=1.0.0 <2.0.0"},
			Alphabetical: &struct {
				Order string `mapstructure:"order" yaml:"order"`
			}{Order: "asc"},
		},
	}
	assert.NoError(t, ValidateImagePolicy(valid))

	invalidRegex := &ImagePolicy{
		FilterTags: &struct {
			Pattern string `mapstructure:"pattern" yaml:"pattern"`
			Extract string `mapstructure:"extract" yaml:"extract"`
		}{
			Pattern: `([`, // invalid regex
			Extract: "",
		},
	}
	assert.Error(t, ValidateImagePolicy(invalidRegex))

	invalidSemver := &ImagePolicy{
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
			}{Range: "not-a-range"},
		},
	}
	assert.Error(t, ValidateImagePolicy(invalidSemver))

	invalidOrder := &ImagePolicy{
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
			}{Order: "up"},
			Alphabetical: &struct {
				Order string `mapstructure:"order" yaml:"order"`
			}{Order: "down"},
		},
	}
	assert.Error(t, ValidateImagePolicy(invalidOrder))
}

func TestValidateConfig(t *testing.T) {
	cfg := &Config{
		Registry: &RegistryConfig{
			DockerHub: &DockerHubConfig{
				ImagePolicy: &ImagePolicy{
					FilterTags: &struct {
						Pattern string `mapstructure:"pattern" yaml:"pattern"`
						Extract string `mapstructure:"extract" yaml:"extract"`
					}{
						Pattern: `^main-[a-z0-9]+-(?P<ts>\\d+)$`,
						Extract: "$ts",
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
				},
			},
			GCR: &GCRConfig{
				ImagePolicy: &ImagePolicy{
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
						}{Range: ">=1.0.0 <2.0.0"},
					},
				},
			},
		},
	}
	assert.NoError(t, ValidateConfig(cfg))

	cfg.Registry.DockerHub.ImagePolicy.FilterTags.Pattern = "(["
	assert.Error(t, ValidateConfig(cfg))

	cfg.Registry.DockerHub.ImagePolicy.FilterTags.Pattern = `^main-[a-z0-9]+-(?P<ts>\\d+)$`
	cfg.Registry.GCR.ImagePolicy.Policy.Semver.Range = "not-a-range"
	assert.Error(t, ValidateConfig(cfg))

	cfg.Registry.GCR.ImagePolicy.Policy.Semver.Range = ">=1.0.0 <2.0.0"
	cfg.Registry.DockerHub.ImagePolicy.Policy.Numerical.Order = "up"
	assert.Error(t, ValidateConfig(cfg))
}

func TestValidateImagePolicy_EdgeCases(t *testing.T) {
	assert.NoError(t, ValidateImagePolicy(nil))
	assert.NoError(t, ValidateImagePolicy(&ImagePolicy{}))
	assert.NoError(t, ValidateImagePolicy(&ImagePolicy{FilterTags: nil, Policy: nil}))
	assert.NoError(t, ValidateImagePolicy(&ImagePolicy{FilterTags: &struct {
		Pattern string `mapstructure:"pattern" yaml:"pattern"`
		Extract string `mapstructure:"extract" yaml:"extract"`
	}{}, Policy: nil}))
}

func TestValidateConfig_EdgeCases(t *testing.T) {
	assert.NoError(t, ValidateConfig(nil))
	assert.NoError(t, ValidateConfig(&Config{}))
	assert.NoError(t, ValidateConfig(&Config{Registry: nil}))
	assert.NoError(t, ValidateConfig(&Config{Registry: &RegistryConfig{}}))
}

func TestValidateConfig_Integration_YAML(t *testing.T) {
	t.Run("Valid YAML with valid policy", func(t *testing.T) {
		yaml := `
registry:
  dockerhub:
    image_policy:
      filterTags:
        pattern: '^main-[a-z0-9]+-(?P<ts>\\d+)$'
        extract: '$ts'
      policy:
        numerical:
          order: desc
        semver:
          range: '>=1.0.0 <2.0.0'
        alphabetical:
          order: asc
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(yaml))
		assert.NoError(t, err)
		var c Config
		err = v.Unmarshal(&c)
		assert.NoError(t, err)
		assert.NoError(t, ValidateConfig(&c))
	})

	t.Run("YAML with invalid regex", func(t *testing.T) {
		yaml := `
registry:
  dockerhub:
    image_policy:
      filterTags:
        pattern: '(['
        extract: ''
      policy:
        numerical:
          order: desc
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(yaml))
		assert.NoError(t, err)
		var c Config
		err = v.Unmarshal(&c)
		assert.NoError(t, err)
		err = ValidateConfig(&c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image_policy.filterTags.pattern")
	})

	t.Run("YAML with invalid semver", func(t *testing.T) {
		yaml := `
registry:
  dockerhub:
    image_policy:
      policy:
        semver:
          range: 'not-a-range'
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(yaml))
		assert.NoError(t, err)
		var c Config
		err = v.Unmarshal(&c)
		assert.NoError(t, err)
		err = ValidateConfig(&c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image_policy.policy.semver.range")
	})

	t.Run("YAML with invalid order", func(t *testing.T) {
		yaml := `
registry:
  dockerhub:
    image_policy:
      policy:
        numerical:
          order: up
        alphabetical:
          order: down
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(yaml))
		assert.NoError(t, err)
		var c Config
		err = v.Unmarshal(&c)
		assert.NoError(t, err)
		err = ValidateConfig(&c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image_policy.policy.numerical.order")
	})

	t.Run("YAML with missing/partial policy", func(t *testing.T) {
		yaml := `
registry:
  dockerhub:
    image_policy:
      filterTags:
        pattern: ''
        extract: ''
      policy: {}
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(strings.NewReader(yaml))
		assert.NoError(t, err)
		var c Config
		err = v.Unmarshal(&c)
		assert.NoError(t, err)
		assert.NoError(t, ValidateConfig(&c))
	})
}
