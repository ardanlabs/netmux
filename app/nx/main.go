package main

import (
	"os"

	"github.com/ardanlabs.com/netmux/app/nx/cli"
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

	log.Infof("main: version %q", build)

	if err := cli.Run(&log); err != nil {
		log.Infof("main: ERROR: %s", err)
		os.Exit(1)
	}
}
