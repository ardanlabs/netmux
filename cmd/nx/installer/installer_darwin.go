package installer

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/service"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
	"go.digitalcircle.com.br/dc/netmux/lib/config"
)

var VarContext = []byte("${CONTEXT}")

var VarNs = []byte("${NS}")

//go:embed netmux.plist
var NxPlist []byte

//go:embed nx.tray.plist
var NxLocalPlist []byte

//go:embed netmux.yaml
var defaultConfig []byte

func AutoInstall(ctx string, ns string, arch string) error {
	var err error
	if arch == "" {
		arch = runtime.GOARCH
	}
	err = Install()
	if err != nil {
		return err
	}
	err = configFileSetup(ctx, ns)
	if err != nil {
		return err
	}

	err = config.Default().Load()
	if err != nil {
		return err
	}

	err = service.Default().Load(config.Default().Fname)
	if err != nil {
		return err
	}

	nxctx := service.Default().CtxByName("default")
	if nxctx == nil {
		return fmt.Errorf("could not load default context - aborting")
	}

	err = service.DefaultK8sController().SetupDeploy(
		context.Background(),
		nxctx,
		ns,
		arch)

	return err
}

func Install() error {
	execName, err := os.Executable()
	if err != nil {
		return err
	}

	fin, err := os.Open(execName)
	if err != nil {
		return err
	}
	_ = os.Remove("/usr/local/bin/nx")

	fout, err := os.OpenFile("/usr/local/bin/nx", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	_, err = io.Copy(fout, fin)
	if err != nil {
		return err
	}
	err = fout.Close()
	if err != nil {
		return err
	}
	_ = os.Remove("/Library/LaunchDaemons/nx.plist")

	err = os.WriteFile("/Library/LaunchDaemons/nx.plist", NxPlist, 0666)
	if err != nil {
		return err
	}
	ret, err := shell.Launchctl.InstallDaemon()
	logrus.Infof(ret)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 10)
	ret, err = shell.Launchctl.StartDaemon()
	logrus.Infof(ret)
	if err != nil {
		logrus.Warnf(err.Error())
	}
	return nil
}

func Uninstall() error {
	ret, err := shell.Launchctl.StopDaemon()
	logrus.Infof(ret)
	if err != nil {
		logrus.Warnf(fmt.Errorf("error stopping daemon: %s, %s", ret, err).Error())
	}

	ret, err = shell.Launchctl.UnistallDaemon()
	logrus.Infof(ret)
	if err != nil {
		logrus.Warnf(fmt.Errorf("error uninstalling daemon: %s, %s", ret, err).Error())
	}

	err = os.Remove("/Library/LaunchDaemons/nx.plist")
	if err != nil {
		return fmt.Errorf("error removing plist file:  %s", err)
	}

	err = os.Remove("/usr/local/bin/nx")
	if err != nil {
		return fmt.Errorf("error nx file:  %s", err)
	}

	return nil
}

func configFileSetup(ctx string, ns string) error {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		username = os.Getenv("USER")
		if username == "" || username == "root" {
			logrus.Warnf("Setup of config cannot be done, undelying user not found")
			return nil
		}
	}

	usr, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("could not resolve underlying user: %s", err.Error())
	}

	cfgFName := filepath.Join(usr.HomeDir, ".netmux.yaml")
	_, err = os.Stat(cfgFName)
	if err == nil {
		logrus.Warnf("A config file already exists - will not create a default one")
		config.Default().Fname = cfgFName
		return nil
	}

	defaultConfig = bytes.Replace(defaultConfig, VarContext, []byte(ctx), -1)

	defaultConfig = bytes.Replace(defaultConfig, VarNs, []byte(ns), -1)

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return err
	}
	err = os.WriteFile(cfgFName, defaultConfig, 0600)
	if err != nil {
		return err
	}
	err = os.Chown(cfgFName, uid, gid)
	if err != nil {
		return err
	}

	config.Default().Fname = cfgFName
	err = config.Default().Save()
	if err != nil {
		return err
	}

	return err
}

func InstallTray() (string, error) {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		username = os.Getenv("USER")
	}
	if username == "root" {
		logrus.Warnf("wont install tray agent to root")
		return "", nil
	}

	aUser, err := user.Lookup(username)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/Library/LaunchAgents/nx.tray.plist", aUser.HomeDir), NxLocalPlist, 0600)
	if err != nil {
		return "", err
	}
	return shell.Launchctl.InstallTrayAgent()
	//cmd.LaunchCtlEnableTrayAgent()
}
