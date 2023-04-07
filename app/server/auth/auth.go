package auth

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type PasswdEntry struct {
	User string `yaml:"user"`
	Hash string `yaml:"hash"`
}

type Passwd struct {
	Entries []PasswdEntry `yaml:"entries"`
}

//go:embed passwd.yaml
var defpasswd []byte

func resolveConfig() (*Passwd, error) {
	fname := os.Getenv("PASSWD")
	if fname == "" {
		fname = "/app/etc/passwd.yaml"
	}
	bs, err := os.ReadFile(fname)
	if err != nil {
		logrus.Warnf("ATTENTION: using default passwd")
		bs = defpasswd
	}
	ret := &Passwd{}
	err = yaml.Unmarshal(bs, ret)
	return ret, err
}

var ErrUserNotFound = fmt.Errorf("user not found")
var ErrAuthError = fmt.Errorf("auth error")
var ErrTokenNotFound = fmt.Errorf("token not found")
