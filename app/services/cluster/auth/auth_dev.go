//go:build dev

package auth

import (
	_ "embed"
	"github.com/sirupsen/logrus"
	"os/user"
)

// Login authenticates a user.
func (a *Auth) Login(name string, passHash string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.Uid, nil
}

// Logout clears the user as being authenticated.
func (a *Auth) Logout(userID string) error {
	return nil
}

// Check validates if the specified user id exists.
func (a *Auth) Check(userID string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.Uid, nil
}
