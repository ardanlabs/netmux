package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/ardanlabs.com/netmux/app/server/grpc"
	"github.com/ardanlabs.com/netmux/app/server/k8s"
	"github.com/ardanlabs/conf/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	//go:embed version
	ver string

	//go:embed semver
	semver string
)

func main() {
	log := logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	if err := run(&log); err != nil {
		log.Infof("startup: ERROR: %s", err)
		os.Exit(1)
	}
}

func run(log *logrus.Logger) error {

	// =========================================================================
	// GOMAXPROCS

	opt := maxprocs.Logger(log.Infof)
	if _, err := maxprocs.Set(opt); err != nil {
		return fmt.Errorf("maxprocs: %w", err)
	}
	log.Infof("startup: GOMAXPROCS: %d", runtime.GOMAXPROCS(0))

	// =========================================================================
	// Configuration

	cfg := struct {
		conf.Version
		Server struct {
			Mode      string `conf:"default:dev-all"`
			Namespace string `conf:"default:default"`
		}
	}{
		Version: conf.Version{
			Build: ver,
			Desc:  "semver: " + semver,
		},
	}

	const prefix = "NX"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// =========================================================================
	// App Starting

	log.Infof("starting service: version %s: semver: %s", ver, semver)
	defer log.Infof("shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Infof("startup: config: %s", out)

	// =========================================================================
	// Start Server

	mns, err := k8s.MyNamespace()
	if err != nil {
		log.Infof("could not determine the namespace: %w", err)
	}
	log.Infof("running from namespace: %s", mns)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		var opts k8s.Opts
		switch cfg.Server.Mode {
		case "dev":
			opts = k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{mns},
			}

		case "dev-all":
			opts = k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{mns},
				All:        true,
			}

		default:
			opts = k8s.Opts{
				//Kubefile:   "~/.kube/k8s.yaml",
				Namespaces: []string{cfg.Server.Namespace},
			}
		}

		if err := k8s.RunDev(ctx, &opts); err != nil {
			log.Infof("k8s.Run in mode %q: %w", cfg.Server.Mode, err)
		}
	}()

	err = grpc.Run()
	if err != nil {
		log.Infof("grpc.Run: %w", err)
	}

	// =========================================================================
	// Stop Server

	cancel()
	wg.Wait()

	return nil
}
