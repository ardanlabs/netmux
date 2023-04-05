package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// TODO: I have seen code that wanted to synchronize access to the config, but
// it wasn't right. I don't know if this is the best way yet to do this, but I
// know it will protect the existing code. I have not been able to clean all
// the code up yet.
var mu sync.RWMutex

// Config represents the configuration needed to run the system.
type Config struct {
	Source string            `yaml:"-"`
	FName  string            `yaml:"fname"`
	Tokens map[string]string `yaml:"tokens"`
}

// Load reads the configuration from disk.
func Load() (Config, error) {
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
func LoadFile(file string) (Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	data, err := os.ReadFile(file)
	if err != nil {
		return Config{}, fmt.Errorf("os.ReadFile: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	return cfg, nil
}

// Save writes the config to storage.
func (cfg Config) Save() error {
	bs, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("yaml.Marshal: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if err := os.WriteFile(cfg.Source, bs, 0600); err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}

	return nil
}

// UpdateFName updates the fname value to disk.
func (cfg Config) UpdateFName(fName string) (Config, error) {
	cfg.FName = fName

	if err := cfg.Save(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
