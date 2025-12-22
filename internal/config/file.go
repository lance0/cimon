// Package config provides configuration file support for cimon (v0.8)
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the cimon.yml configuration file structure
type FileConfig struct {
	Repositories []string `yaml:"repositories"` // owner/repo format
}

// LoadConfigFile loads configuration from a YAML file.
// Returns nil, nil if the file doesn't exist (not an error).
func LoadConfigFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file is OK
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", path, err)
	}

	return &cfg, nil
}

// ToRepoSpecs converts FileConfig repositories to RepoSpec slice
func (f *FileConfig) ToRepoSpecs() ([]RepoSpec, error) {
	if f == nil || len(f.Repositories) == 0 {
		return nil, nil
	}

	var specs []RepoSpec
	for _, r := range f.Repositories {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		parts := strings.SplitN(r, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid repo format %q in config file: expected owner/repo", r)
		}
		specs = append(specs, RepoSpec{Owner: parts[0], Repo: parts[1]})
	}

	return specs, nil
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	return "cimon.yml"
}
