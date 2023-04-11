package main

import (
	"os"
	"os/user"

	"github.com/ardanlabs.com/netmux/app/ui/tray/tray"
	"github.com/sirupsen/logrus"
)

// build is the git version of this program. It is set using build flags in the makefile.
var build = "develop"

func main() {
	log := logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	if err := run(&log); err != nil {
		log.Infof("main: ERROR: %s", err)
		os.Exit(1)
	}
}

func run(log *logrus.Logger) error {
	log.Infof("main: version %q", build)

	user, err := user.Current()
	if err != nil {
		return err
	}

	if err := tray.Run(log, user); err != nil {
		return err
	}

	return nil
}
