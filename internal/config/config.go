// Package config loads and parses Ph.Sh_url's config.yaml, including
// stripping unedited placeholder API keys.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	dirName  = "Ph.Sh_url"
	fileName = "config.yaml"
)

// Config holds per-source API keys loaded from config.yaml.
type Config struct {
	VirusTotalKeys []string `yaml:"virustotal"`
	AlienVaultKeys []string `yaml:"alienvault"`
	HudsonRockKeys []string `yaml:"hudsonrock"`
}

// LoadConfig reads config.yaml from the user's config directory, creating a
// default template file (and returning an error asking the user to edit it)
// if none exists yet.
func LoadConfig(silent bool) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", dirName, fileName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if !silent {
			log.Printf("Configuration file not found at %s", configPath)
		}
		if err := CreateDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("could not create default config file: %w", err)
		}
		return nil, fmt.Errorf("a new configuration file has been created at %s. Please edit it to add your API keys", configPath)
	}

	if !silent {
		log.Printf("Loading configuration from %s", configPath)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	cfg.VirusTotalKeys = filterPlaceholderKeys(cfg.VirusTotalKeys)
	cfg.AlienVaultKeys = filterPlaceholderKeys(cfg.AlienVaultKeys)
	cfg.HudsonRockKeys = filterPlaceholderKeys(cfg.HudsonRockKeys)

	return &cfg, nil
}

// filterPlaceholderKeys removes empty keys and unedited placeholder values
// (e.g. "YOUR_VT_API_KEY_1") left over from the default config template,
// so sources fall back to keyless mode instead of sending invalid keys.
func filterPlaceholderKeys(keys []string) []string {
	var valid []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" || strings.HasPrefix(k, "YOUR_") {
			continue
		}
		valid = append(valid, k)
	}
	return valid
}

// CreateDefaultConfig writes a template config.yaml at path.
func CreateDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	defaultConfig := `# Configuration file for Ph.Sh_URL
virustotal:
  - "YOUR_VT_API_KEY_1"
alienvault:
  - "YOUR_OTX_API_KEY_1"
hudsonrock:
  - "YOUR_HUDSONROCK_API_KEY_1"
`

	return os.WriteFile(path, []byte(defaultConfig), 0644)
}
