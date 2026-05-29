package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	appName    = "openshell-delegate"
	configFile = "config.yaml"
)

// Config is the openshell-delegate configuration.
type Config struct {
	// Image is the default container image for sandboxes.
	Image string `yaml:"image"`

	// Policy is the path to the default network policy YAML.
	Policy string `yaml:"policy"`

	// Provider is the OpenShell provider name (e.g., "opencode-vertex").
	Provider string `yaml:"provider"`

	// Env is a list of environment variable names to forward from the host.
	Env []string `yaml:"env"`

	// StaticEnv is a map of environment variables to set to fixed values.
	StaticEnv map[string]string `yaml:"static_env"`

	// Uploads is a list of files to upload into the sandbox at creation time.
	Uploads []Upload `yaml:"uploads"`
}

// Upload describes a file to upload from host to sandbox.
type Upload struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
}

// ConfigDir returns the configuration directory path.
func ConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appName), nil
}

// Load reads the configuration file and returns the config.
// Returns a zero-value Config if the file doesn't exist.
func Load() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return &Config{}, nil
	}

	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("could not read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	return &cfg, nil
}

// ExpandPath expands ~ in file paths.
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
