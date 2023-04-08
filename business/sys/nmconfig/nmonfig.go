// Package nmconfig manages the netmux configuration file.
package nmconfig

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// TODO: Instead of using a filename, let's use a io.ReadWriter.
// TODO: We can't read these env variables from here.

// Set of error variables.
var (
	ErrNotFound = errors.New("not found")
)

// Config represents the configuration needed to run the system.
type Config struct {
	source string            `yaml:"-"`
	mu     sync.RWMutex      `yaml:"-"`
	fName  string            `yaml:"fname"`
	tokens map[string]string `yaml:"tokens"`
}

// Load reads the configuration from disk.
func Load() (*Config, error) {
	file := configFile
	if os.Getenv("KUBERNETES_PORT") != "" {
		switch {
		case os.Getenv("CONFIG") != "":
			file = os.Getenv("CONFIG")
		default:
			file = "/app/persistence/netmux.yaml"
			os.MkdirAll("/app/persistence", os.ModePerm)
		}
	}

	return LoadFile(file)
}

// LoadFile reads the specified configuration from disk.
func LoadFile(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	return &cfg, nil
}

// UpdateFileName updates the fname value to disk.
func (c *Config) UpdateFileName(fName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.fName = fName

	if err := c.save(); err != nil {
		return err
	}

	return nil
}

// AddToken adds a new token to the config.
func (c *Config) AddToken(key string, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[key] = value

	if err := c.save(); err != nil {
		return err
	}

	return nil
}

// DeleteToken removes a new token from the config.
func (c *Config) DeleteToken(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, key)

	if err := c.save(); err != nil {
		return err
	}

	return nil
}

// FileName returns the file name from the config.
func (c *Config) FileName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.fName
}

// Token looks up a token from the configuration.
func (c *Config) Token(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.tokens[key]
	if !exists {
		return "", ErrNotFound
	}

	return value, nil
}

// =============================================================================

// Save writes the config to storage.
func (c *Config) save() error {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("yaml.Marshal: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.WriteFile(c.source, bs, 0600); err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}

	return nil
}
