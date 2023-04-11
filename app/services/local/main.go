package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"syscall"

	"github.com/ardanlabs.com/netmux/app/services/local/cluster"
	"github.com/ardanlabs.com/netmux/app/services/local/grpc"
	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
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
			ConfigFileName string `conf:"default:/etc/netmux.yaml"`
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

	cluster, err := cluster.Start(bridge.DirectionForward)
	if err != nil {
		return fmt.Errorf("user.Current: %w", err)
	}
	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("user.Current: %w", err)
	}

	grpc, err := grpc.Start(log, user, cluster)
	if err != nil {
		return fmt.Errorf("grpc.Start: %w", err)
	}

	// =========================================================================
	// Stop Services

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	grpc.Shutdown()
	cluster.Shutdown()

	return nil
}
