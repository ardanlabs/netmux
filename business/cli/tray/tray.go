// Package tray provides support for running the trap application.
package tray

import "C"
import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
	"github.com/gen2brain/beeep"
	"github.com/sirupsen/logrus"
)

//go:embed duck-w.png
var logo []byte

const (
	eventTypeDisconnected = "disconnected"
	eventTypeConnected    = "connected"
	eventKATimeOut        = "katimeout"
)

// =============================================================================

// Run starts the tray application.
func Run(log *logrus.Logger, user *user.User) error {
	fyneApp := app.New()
	desktopApp, ok := fyneApp.(desktop.App)
	if !ok {
		return errors.New("fyneApp to desktop app conversation")
	}

	// -------------------------------------------------------------------------

	// TODO: Not sure what this client is doing? We don't Exit it and it's never
	//       referenced again.

	cln, err := agent.NewClient("", "")
	if err != nil {
		beeep.Alert("tray", fmt.Sprintf("ERROR: %s", err.Error()), "")
		return fmt.Errorf("agent.NewClient: %w", err)
	}

	if _, err = cln.Load(context.Background(), &agent.StringMsg{Msg: filepath.Join(user.HomeDir, ".netmux.yaml")}); err != nil {
		beeep.Alert("tray", fmt.Sprintf("ERROR: %s", err.Error()), "")
		return fmt.Errorf("client.Load: %w", err)
	}

	// -------------------------------------------------------------------------

	mis := []*fyne.MenuItem{}

	mwFunc := func() {
		ex, err := os.Executable()
		if err != nil {
			log.Infof("tray: os.Executable: %w", err)
			beeep.Alert("tray: os.Executable", fmt.Sprintf("ERROR: %s", err.Error()), "")
			return
		}

		cmd := exec.Command(ex, "webview")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			log.Infof("tray: cmd.Start: %w", err)
			beeep.Alert("tray: cmd.Start", fmt.Sprintf("ERROR: %s", err.Error()), "")
		}
	}
	mis = append(mis, fyne.NewMenuItem("Main Window", mwFunc))

	// TODO: This next function makes no sense. You create a connection and then
	//       immediately Exit.

	raFunc := func() {
		client, err := agent.NewClient("", "")
		if err != nil {
			log.Infof("tray: agent.NewClient: %w", err)
			beeep.Alert("tray", fmt.Sprintf("ERROR: %s", err.Error()), "")
			return
		}

		client.Exit(context.Background(), &agent.Noop{})
	}
	mis = append(mis, fyne.NewMenuItem("Restart Agent", raFunc))

	desktopApp.SetSystemTrayMenu(fyne.NewMenu("Netmux", mis...))
	desktopApp.SetSystemTrayIcon(fyne.NewStaticResource("logo", logo))

	// -------------------------------------------------------------------------

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	var mu sync.RWMutex
	var client agent.AgentClient

	go func() {
		defer wg.Done()

		for {
			log.Info("tray: waiting to connect to the agent...")

			time.Sleep(time.Second)

			if ctx.Err() != nil {
				log.Info("tray: context deadlined")
				return
			}

			mu.Lock()
			{
				log.Info("tray: connecting to the agent...")
				client, err = agent.NewClient("", "")
			}
			mu.Unlock()

			if err != nil {
				log.Infof("tray: agent.NewClient: %w", err)
				continue
			}

			if err := processEvents(ctx, log, client); err != nil {
				log.Infof("tray: %s", err)
				return
			}

			if _, err := client.Exit(ctx, &agent.Noop{}); err != nil {
				log.Infof("tray: client.Exit: %w", err)
			}

		}
	}()

	// -------------------------------------------------------------------------

	fyneApp.Lifecycle().SetOnStarted(func() {
		go func() {
			time.Sleep(200 * time.Millisecond)
			setActivationPolicy()
		}()
	})

	fyneApp.Run()

	// -------------------------------------------------------------------------

	cancel()

	mu.RLock()
	{
		_, err = client.Exit(ctx, &agent.Noop{})
	}
	mu.RUnlock()

	if err != nil {
		log.Infof("tray: client.Exit: %w", err)
	}

	wg.Wait()

	return nil
}

// =============================================================================

func processEvents(ctx context.Context, log *logrus.Logger, client agent.AgentClient) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Info("tray: register to receive events")
		events, err := client.Events(context.Background(), &agent.Noop{})
		if err != nil {
			log.Infof("tray: client.Events: %w", err)
			return nil
		}

		for {
			eventMsg, err := events.Recv()
			if err != nil {
				log.Infof("tray: events.Recv: %w", err)
				beeep.Alert("tray", fmt.Sprintf("Error receiving event: %s", err.Error()), "")
				break
			}

			switch eventMsg.MsgType {
			case eventTypeConnected:
				beeep.Alert("tray", fmt.Sprintf("Connected to: %s", eventMsg.Ctx), "")

			case eventTypeDisconnected:
				beeep.Alert("tray", fmt.Sprintf("Disconnected from: %s", eventMsg.Ctx), "")
			}
		}
	}
}
