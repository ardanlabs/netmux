package main

import (
	_ "embed"

	"github.com/ardanlabs.com/netmux/app/nx/cli"
	"github.com/ardanlabs.com/netmux/app/nx/service"
	"github.com/sirupsen/logrus"
)

//go:embed version
var ver string

//go:embed semver
var semver string

func run() error {

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})
	service.Ver = ver
	service.Semver = semver

	return cli.Run()
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
