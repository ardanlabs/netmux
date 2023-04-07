//go:build !dev

package auth

import (
	_ "embed"
	"time"

	"github.com/ardanlabs.com/netmux/foundation/config"
	"github.com/ardanlabs.com/netmux/foundation/hash"
	"k8s.io/apimachinery/pkg/util/rand"
)

func Login(user string, pass string) (string, error) {
	pwd, err := resolveConfig()
	if err != nil {
		return "", err
	}

	for _, e := range pwd.Entries {
		if e.User == user {
			if err := hash.Decode(e.Hash, pass); err != nil {
				return "", ErrAuthError
			}

			cfg, err := config.Load()
			if err != nil {
				return "", err
			}

			rand.Seed(time.Now().UnixMilli())
			uid := rand.String(32)

			cfg.Tokens[uid] = user
			if err := cfg.Save(); err != nil {
				return "", err
			}

			return uid, nil
		}
	}

	return "", ErrUserNotFound
}

func Logout(token string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	delete(cfg.Tokens, token)
	return cfg.Save()
}

func Check(token string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	user, ok := cfg.Tokens[token]
	if ok && user != "" {
		return user, nil
	}

	return "", ErrTokenNotFound
}
