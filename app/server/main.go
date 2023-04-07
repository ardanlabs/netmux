package main

import (
	"context"
	_ "embed"
	"os"

	"github.com/ardanlabs.com/netmux/app/server/grpc"
	"github.com/ardanlabs.com/netmux/app/server/k8s"
	"github.com/sirupsen/logrus"
)

//go:embed version
var ver string

//go:embed semver
var semver string

func main() {
	logrus.Infof("netmux server starting - ver: %s - %s", semver, ver)
	logrus.SetLevel(logrus.TraceLevel)

	mns, err := k8s.MyNamespace()
	if err != nil {
		logrus.Warnf("Could not determine own namespace")
	} else {
		logrus.Infof("Running from namespace: %s", mns)
	}

	mode := os.Getenv("MODE")
	logrus.Infof("Running server in mode: %s", mode)
	go func() {
		var err error
		switch mode {
		case "dev":
			err = k8s.RunDev(context.Background(), &k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{mns},
			})
		case "dev-all":
			err = k8s.RunDev(context.Background(), &k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{mns},
				All:        true,
			})

		default:
			logrus.Infof("Using namespace: %s", os.Getenv("NS"))
			err = k8s.Run(context.Background(), &k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{os.Getenv("NS")},
			})

		}
		if err != nil {
			panic(err)
		}
	}()

	err = grpc.Run()
	if err != nil {
		panic(err)
	}
}
