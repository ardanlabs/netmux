package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

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
	// K8s Configuration

	var k8sCfg k8s.Config
	switch cfg.Server.Mode {
	case "dev":
		k8sCfg = k8s.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces: []string{k8s.Namespace()},
		}

	case "dev-all":
		k8sCfg = k8s.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces:  []string{k8s.Namespace()},
			AllServices: true,
		}

	default:
		k8sCfg = k8s.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces: []string{cfg.Server.Namespace},
		}
	}

	// =========================================================================
	// Start Server

	server, err := grpc.Start(log)
	if err != nil {
		log.Infof("grpc.Start: %w", err)
	}

	k8s, err := k8s.Start(log, server, k8sCfg)
	if err != nil {
		log.Infof("main: k8s.Start: mode[%s]: %w", cfg.Server.Mode, err)
	}

	// =========================================================================
	// Stop Server

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	k8s.Shutdown()
	server.Shutdown()

	return nil
}
