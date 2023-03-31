//go:build dev

package auth

import (
	_ "embed"
	"github.com/sirupsen/logrus"
	"os/user"
)

func init() {
	logrus.Warnf("RUNNING AUTH DEV MODE!!!")
}

func Login(username string, pass string) (string, error) {

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.Uid, nil
}

func Logout(token string) error {
	return nil
}

func Check(token string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.Uid, nil
}
