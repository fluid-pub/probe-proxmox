package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fluid/probes/core"

	"gopkg.in/yaml.v3"
)

// Config is the Proxmox VE probe configuration.
type Config struct {
	Probe        core.ProbeConfig         `yaml:"probe"`
	Proxmox      ProxmoxConfig            `yaml:"proxmox"`
	Data         core.DataConfig          `yaml:"data"`
	State        core.StateConfig         `yaml:"state"`
	Controlplane *core.ControlplaneConfig `yaml:"controlplane,omitempty"`
	// Filled from config/schema.yml (usable_in_rag); not serialized.
	RAGFieldAllowlist core.RAGFieldSet `yaml:"-"`
}

// ProxmoxConfig holds Proxmox VE API settings.
// api_token must be the full PVE API token value: user@realm!token_id=secret
type ProxmoxConfig struct {
	APIURL                string `yaml:"api_url"`
	APIToken              string `yaml:"api_token"`
	Timeout               string `yaml:"timeout,omitempty"`
	MaxRetries            int    `yaml:"max_retries,omitempty"`
	TLSInsecureSkipVerify bool   `yaml:"tls_insecure_skip_verify,omitempty"`
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read configuration file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}

	if err := config.resolveEnvironmentVariables(); err != nil {
		return nil, fmt.Errorf("resolve environment variables: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	schemaPath := filepath.Join(filepath.Dir(configPath), "schema.yml")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schema.yml required (same directory as config) to validate fields.*.rag: %w", err)
	}
	allowed, err := core.ParseRAGFieldSetFromSchemaYAML(schemaData)
	if err != nil {
		return nil, fmt.Errorf("schema.yml (RAG): %w", err)
	}
	if err := core.ValidateRAGEntityFields(config.Data.Entities, allowed); err != nil {
		return nil, err
	}
	config.RAGFieldAllowlist = allowed

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Probe.Name == "" {
		return fmt.Errorf("probe name is missing")
	}

	if c.Proxmox.APIURL == "" {
		return fmt.Errorf("proxmox api_url is missing")
	}

	if c.Proxmox.APIToken == "" {
		return fmt.Errorf("proxmox api_token is missing")
	}

	if c.Proxmox.Timeout == "" {
		c.Proxmox.Timeout = "30s"
	}

	if c.Proxmox.MaxRetries <= 0 {
		c.Proxmox.MaxRetries = 3
	}

	if len(c.Data.Entities) == 0 {
		return fmt.Errorf("at least one entity must be configured")
	}

	for i, entity := range c.Data.Entities {
		if entity.Name == "" {
			return fmt.Errorf("entity at index %d has no name", i)
		}
		if entity.RefreshInterval == "" {
			return fmt.Errorf("entity %s has no refresh_interval", entity.Name)
		}
	}

	if c.State.Dir == "" {
		return fmt.Errorf("state directory is missing")
	}

	if c.State.CleanupInterval <= 0 {
		c.State.CleanupInterval = 1
	}

	return nil
}

func (c *Config) resolveEnvironmentVariables() error {
	if c.Proxmox.APIURL != "" {
		if resolved, isEnvVar := resolveEnvVar(c.Proxmox.APIURL); isEnvVar {
			if resolved == "" {
				return fmt.Errorf("environment variable for proxmox api_url is not defined")
			}
			c.Proxmox.APIURL = resolved
		}
	}

	if c.Proxmox.APIToken != "" {
		if resolved, isEnvVar := resolveEnvVar(c.Proxmox.APIToken); isEnvVar {
			if resolved == "" {
				return fmt.Errorf("environment variable for proxmox api_token is not defined")
			}
			c.Proxmox.APIToken = resolved
		}
	}

	if c.Controlplane != nil {
		if c.Controlplane.BaseURL != "" {
			if resolved, isEnvVar := resolveEnvVar(c.Controlplane.BaseURL); isEnvVar {
				if resolved == "" {
					log.Printf("Warning: environment variable not defined for base_url, controlplane disabled")
					c.Controlplane = nil
					return nil
				}
				c.Controlplane.BaseURL = resolved
			}
		}

		if c.Controlplane.Parameters == nil {
			log.Printf("Warning: controlplane parameters missing, controlplane disabled")
			c.Controlplane = nil
			return nil
		}

		if c.Controlplane.Parameters.OrganizationUUID != "" {
			if resolved, isEnvVar := resolveEnvVar(c.Controlplane.Parameters.OrganizationUUID); isEnvVar && resolved != "" {
				c.Controlplane.Parameters.OrganizationUUID = resolved
			}
		}
		if c.Controlplane.Parameters.Token != "" {
			if resolved, isEnvVar := resolveEnvVar(c.Controlplane.Parameters.Token); isEnvVar && resolved != "" {
				c.Controlplane.Parameters.Token = resolved
			}
		}
		if c.Controlplane.Parameters.OrganizationUUID == "" || c.Controlplane.Parameters.Token == "" {
			log.Printf("Warning: controlplane parameters incomplete, controlplane disabled")
			c.Controlplane = nil
		}
	}

	return nil
}

func resolveEnvVar(value string) (string, bool) {
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return value, false
	}
	envVar := strings.TrimPrefix(strings.TrimSuffix(value, "}"), "${")
	return os.Getenv(envVar), true
}

func (c *Config) GetProbeName() string                      { return c.Probe.Name }
func (c *Config) GetProbeVersion() string                   { return c.Probe.Version }
func (c *Config) GetStateDir() string                       { return c.State.Dir }
func (c *Config) GetCleanupInterval() int                   { return c.State.CleanupInterval }
func (c *Config) GetEntities() []core.EntityConfig          { return c.Data.Entities }
func (c *Config) GetControlplane() *core.ControlplaneConfig { return c.Controlplane }
