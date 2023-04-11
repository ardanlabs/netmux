//go:build !dev

package auth

import (
	_ "embed"
	"time"

	"github.com/ardanlabs.com/netmux/foundation/hash"
	"k8s.io/apimachinery/pkg/util/rand"
)

// Login authenticates a user.
func (a *Auth) Login(name string, passHash string) (string, error) {
	usr, exists := a.users[name]
	if !exists {
		return "", ErrUserNotFound
	}

	if err := hash.Decode(usr.PassHash, passHash); err != nil {
		return "", ErrAuthError
	}

	rand.Seed(time.Now().UnixMilli())
	userID := rand.String(32)

	if err := a.nmconfig.AddToken(userID, usr.Name); err != nil {
		return "", err
	}

	return userID, nil
}

// Logout clears the user as being authenticated.
func (a *Auth) Logout(userID string) error {
	if err := a.nmconfig.DeleteToken(userID); err != nil {
		return err
	}

	return nil
}

// Check validates if the specified user id exists.
func (a *Auth) Check(userID string) (string, error) {
	usr, err := a.nmconfig.Token(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	return usr, nil
}
