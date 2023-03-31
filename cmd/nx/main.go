package main

import (
	_ "embed"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/cli"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/service"
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
