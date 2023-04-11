//go:build !dev

package auth

import (
	_ "embed"
	"go.digitalcircle.com.br/dc/netmux/lib/argon2"
	"go.digitalcircle.com.br/dc/netmux/lib/config"
	"k8s.io/apimachinery/pkg/util/rand"
	"time"
)

func Login(user string, pass string) (string, error) {
	mx.Lock()
	defer mx.Unlock()
	pwd, err := resolveConfig()
	if err != nil {
		return "", err
	}
	for _, e := range pwd.Entries {
		if e.User == user {
			ok, err := argon2.ComparePasswordAndHash(pass, e.Hash)
			if err != nil {
				return "", err
			}
			if !ok {
				return "", ErrAuthError
			}

			rand.Seed(time.Now().UnixMilli())
			uid := rand.String(32)
			config.Default().Tokens[uid] = user
			err = config.Default().Save()
			return uid, err

		}
	}
	return "", ErrUserNotFound
}

func Logout(token string) error {
	mx.Lock()
	defer mx.Unlock()
	delete(config.Default().Tokens, token)
	return config.Default().Save()
}

func Check(token string) (string, error) {
	mx.RLock()
	defer mx.RUnlock()
	user, ok := config.Default().Tokens[token]
	if ok && user != "" {
		return user, nil
	}
	return "", ErrTokenNotFound
}
