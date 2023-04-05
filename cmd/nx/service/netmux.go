package service

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
	"gopkg.in/yaml.v3"
)

type Status string
type RuntimeType string

var Ver string
var Semver string

const (
	StatusDisconnected Status      = "disconnected"
	StatusConnecting   Status      = "connecting"
	StatusAvailable    Status      = "available"
	StatusDisabled     Status      = "disabled"
	StatusStarting     Status      = "starting"
	StatusStarted      Status      = "started"
	StatusRunning      Status      = "running"
	StatusStopping     Status      = "stopping"
	StatusStopped      Status      = "stopped"
	StatusLoading      Status      = "loading"
	StatusError        Status      = "error"
	RuntimeKubernetes  RuntimeType = "kubernetes"

	Sock = "/tmp/netmux.sock"
)

var ErrCouldNotReadConfig = fmt.Errorf("could not read config")

var defaultCfg = new(Netmux)

func Reset() {
	defaultCfg.Stop()
	defaultCfg = new(Netmux)
}

func Default() *Netmux {
	return defaultCfg
}

type Auth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Netmux struct {
	Semver             string     `yaml:"semver"`
	Version            string     `yaml:"version"`
	Kubectl            string     `yaml:"kubectl"`
	Dashboard          string     `yaml:"dashboard"`
	InsecureSkipVerify bool       `yaml:"insecureSkipVerify"`
	Iface              string     `yaml:"iface"`
	Contexts           []*Context `yaml:"contexts"`
	IpAliasMask        string     `yaml:"ipaliasmask"`
	Auth               Auth       `yaml:"auth"`

	// Transient info
	ctx    context.Context
	cancel func()
	// tokens db.DB[string]   // TODO: Not being used?

	SourceFile string `yaml:"-"`
	Username   string `yaml:"-"`
	Hostname   string `yaml:"-"`
	Userhome   string `yaml:"-"`
	HostOS     string `yaml:"-"`
	HostArch   string `yaml:"hostArch"`
	WorkingDir string `yaml:"-"`
	Status     Status `yaml:"-"`
}

func (c *Netmux) Prepare(loggedUserName string, s string) error {
	var err error

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	actUser, err := user.Current()
	if err != nil {
		return err
	}
	var runningUser *user.User
	if actUser.Username == "root" {

		runningUser, err = user.Lookup(os.Getenv("SUDO_USER"))

		if runningUser == nil {

			loggedUser, err := user.Lookup(loggedUserName)
			if err != nil {
				return err
			}
			runningUser = loggedUser

		}
	} else {
		runningUser = actUser
	}

	if s == "" {
		s = os.Getenv("NETMUX_CONFIG")
	}

	if s == "" {
		for _, s := range []string{
			filepath.Join("netmux"),
			filepath.Join("netmux.yaml"),
			filepath.Join("netmux.yml"),
			filepath.Join(runningUser.HomeDir, ".netmux"),
			filepath.Join(runningUser.HomeDir, ".netmux.yaml"),
			filepath.Join(runningUser.HomeDir, ".netmux.yml"),
			filepath.Join(runningUser.HomeDir, ".config/netmux"),
			filepath.Join(runningUser.HomeDir, ".config/netmux.yaml"),
			filepath.Join(runningUser.HomeDir, ".config/netmux.yml"),
			filepath.Join(runningUser.HomeDir, "netmux"),
			filepath.Join(runningUser.HomeDir, "netmux.yaml"),
			filepath.Join(runningUser.HomeDir, "netmux.yml"),
			filepath.Join("/etc/netmux"),
			filepath.Join("/etc/netmux.yaml"),
			filepath.Join("/etc/netmux.yml"),
		} {
			bs, err := os.ReadFile(s)
			if err == nil {
				err = yaml.Unmarshal(bs, c)
				if err != nil {
					return fmt.Errorf("error loading %s: %s", s, err.Error())
				}
				c.SourceFile = s
				break
			}
		}
	} else {
		s = strings.Replace(s, "~", runningUser.HomeDir, 1)
		bs, err := os.ReadFile(s)
		if err == nil {
			err = yaml.Unmarshal(bs, c)
			if err != nil {
				return fmt.Errorf("error loading %s: %s", s, err.Error())
			}
			c.SourceFile = s
		}
	}

	if c.SourceFile == "" {
		return ErrCouldNotReadConfig
	}

	c.Username = runningUser.Username
	c.Userhome = runningUser.HomeDir
	c.Hostname, err = os.Hostname()
	c.HostOS = runtime.GOOS
	c.HostArch = runtime.GOARCH
	c.WorkingDir = wd
	for _, v := range c.Contexts {
		err = v.Prepare(c)
		if err != nil {
			return err
		}
	}
	if c.IpAliasMask == "" {
		c.IpAliasMask = "10.1.0.%v"
	}
	c.SourceFile, err = filepath.Abs(c.SourceFile)
	if err != nil {
		return err
	}
	defaultCfg = c

	c.Semver = Semver
	c.Version = Ver

	c.Status = StatusAvailable

	return nil
}

func (c *Netmux) CtxByName(n string) *Context {

	for _, v := range c.Contexts {
		if v.Name == n {
			return v
		}
	}
	return nil
}

func (c *Netmux) ResetCounters() {
	for _, v := range c.Contexts {
		v.ResetCounters()
	}
}
func (c *Netmux) Stop() error {
	c.Status = StatusStopping
	for _, v := range c.Contexts {
		err := v.Stop()
		if err != nil {
			logrus.Warnf("Error stopping context: %s: %s", v.Name, err.Error())
		}
	}
	c.Status = StatusStopped
	return nil
}

func (c *Netmux) Load(f string) error {
	var err error
	c.Status = StatusLoading
	loggedUser, err := shell.Who.ConsoleUser()
	if err != nil {
		return err
	}

	err = c.Prepare(loggedUser, f)

	return err
}

func (c *Netmux) Start(username string, fname string, ctxs []string) (err error) {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.Status = StatusStarting
	defer func() {
		if err != nil {
			c.Status = StatusError
		}
	}()

	err = c.Prepare(username, fname)
	if err != nil {
		return err
	}

	logrus.Infof("Using config: " + Default().SourceFile)

	err = c.loadContexts(ctxs)

	c.Status = StatusRunning
	return
}
func (c *Netmux) loadContexts(ctxs []string) error {

	has := func(s string) bool {
		for _, v := range ctxs {
			if s == v {
				return true
			}
		}
		return false
	}

	for _, actx := range c.Contexts {
		if !actx.Enabled {
			logrus.Debugf("Ctx %s is not enabled, skipping", actx.Name)
			continue
		}
		if len(ctxs) == 0 || ctxs[0] == "*" || has(actx.Name) {
			var chControl = make(chan struct{})
			var chErr = make(chan error)
			go func() {
				err := actx.Start(c.ctx, chControl)
				chErr <- err
			}()
			select {
			case <-chControl:
			case err := <-chErr:
				return err
			case <-time.After(time.Second * 30):
				return fmt.Errorf("timeout setting up context: %s", actx.Name)
			}

		} else {
			err := actx.markContextDisabled()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
