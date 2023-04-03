package config

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func trace(s string, p ...interface{}) {
	logrus.Tracef("Config: %s", fmt.Errorf(s, p...))
}

type Config struct {
	Src    string            `yaml:"-"`
	Fname  string            `yaml:"fname"`
	Tokens map[string]string `yaml:"tokens"`
}

func (c *Config) Save() error {
	trace("Saving config")
	bs, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Src, bs, 0600)
}

func (c *Config) Load() error {
	trace("Loading config")
	src := c.Src
	bs, err := os.ReadFile(c.Src)
	if err != nil {
		logrus.Warnf("Couldnt load config - using defaults: %s", err.Error())
		return nil
	}
	err = yaml.Unmarshal(bs, c)
	c.Src = src
	return err
}

func New() (*Config, error) {
	ret := Config{
		Src:    Fname,
		Fname:  "",
		Tokens: make(map[string]string),
	}

	if os.Getenv("KUBERNETES_PORT") != "" {
		if os.Getenv("CONFIG") != "" {
			ret.Src = os.Getenv("CONFIG")
			logrus.Infof("Using config from: %s", ret.Src)
		} else {
			ret.Src = "/app/persistence/netmux.yaml"
			err := os.MkdirAll("/app/persistence", os.ModePerm)
			if err != nil {
				return nil, err
			}
			logrus.Infof("Using config from: %s", ret.Src)
		}

	}

	return &ret, nil
}

var def *Config

func Default() *Config {
	if def == nil {
		var err error
		def, err = New()
		if err != nil {
			panic(err)
		}
		err = def.Load()
		if err != nil {
			panic(err)
		}
	}
	return def
}
