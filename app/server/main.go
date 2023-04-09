package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ardanlabs.com/netmux/app/server/auth"
	"github.com/ardanlabs.com/netmux/app/server/grpc"
	"github.com/ardanlabs.com/netmux/app/server/monitor"
	"github.com/ardanlabs.com/netmux/business/sys/nmconfig"
	"github.com/ardanlabs/conf/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
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

	// =========================================================================
	// GOMAXPROCS

	opt := maxprocs.Logger(log.Infof)
	if _, err := maxprocs.Set(opt); err != nil {
		return fmt.Errorf("maxprocs: %w", err)
	}
	log.Infof("main: GOMAXPROCS: %d", runtime.GOMAXPROCS(0))

	// =========================================================================
	// Configuration

	cfg := struct {
		conf.Version
		Server struct {
			Mode      string `conf:"default:dev-all"`
			Namespace string `conf:"default:default"`
		}
		Auth struct {
			PasswordFile string `conf:"default:zarf/users/default_users.yaml"`
		}
	}{
		Version: conf.Version{
			Build: build,
			Desc:  "copyright information here",
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

	log.Infof("main: version %q", build)
	defer log.Infof("main: shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Infof("main: config: %s", out)

	// =========================================================================
	// Start Service

	nmconfig, err := nmconfig.Load()
	if err != nil {
		return fmt.Errorf("nmconfig.Load: %w", err)
	}

	a, err := auth.New(cfg.Auth.PasswordFile, nmconfig)
	if err != nil {
		return fmt.Errorf("auth.New: %w", err)
	}

	service, err := grpc.Start(log, a)
	if err != nil {
		return fmt.Errorf("grpc.Start: %w", err)
	}

	// =========================================================================
	// Start Monitor

	var mntCfg monitor.Config
	switch cfg.Server.Mode {
	case "dev":
		mntCfg = monitor.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces: []string{monitor.Namespace()},
		}

	case "dev-all":
		mntCfg = monitor.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces:  []string{monitor.Namespace()},
			AllServices: true,
		}

	default:
		mntCfg = monitor.Config{
			//Kubefile:   "~/.kube/k8s.yaml",
			Namespaces: []string{cfg.Server.Namespace},
		}
	}

	monitor, err := monitor.Start(log, service, mntCfg)
	if err != nil {
		log.Infof("main: k8s.Start: mode[%s]: %w", cfg.Server.Mode, err)
	}

	// =========================================================================
	// Stop Services

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	monitor.Shutdown()
	service.Shutdown()

	return nil
}
