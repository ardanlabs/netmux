// Package auth provides support for authentication.
package auth

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/ardanlabs.com/netmux/business/sys/nmconfig"
	"gopkg.in/yaml.v2"
)

// Set of error variables.
var (
	ErrUserNotFound = fmt.Errorf("user not found")
	ErrAuthError    = fmt.Errorf("auth error")
)

// User represents a user and their password hash.
type User struct {
	Name     string `yaml:"user"`
	PassHash string `yaml:"hash"`
}

// =============================================================================

// Auth represents support for authentication.
type Auth struct {
	nmconfig *nmconfig.Config
	users    map[string]User
}

// New constructs an auth value for use.
func New(fName string, nmconfig *nmconfig.Config) (*Auth, error) {
	yaml, err := os.ReadFile(fName)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	return Load(yaml, nmconfig)
}

// Load takes a pre-read config and constructs an auth value for use.
func Load(data []byte, nmconfig *nmconfig.Config) (*Auth, error) {
	var users struct {
		Entries []User `yaml:"entries"`
	}
	if err := yaml.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	m := make(map[string]User)
	for _, password := range users.Entries {
		m[password.Name] = password
	}

	ath := &Auth{
		users:    m,
		nmconfig: nmconfig,
	}

	return ath, nil
}
