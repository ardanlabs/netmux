package tray

import "C"
import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
	"github.com/gen2brain/beeep"
	"github.com/sirupsen/logrus"
)

const (
	eventTypeDisconnected = "disconnected"
	eventTypeConnected    = "connected"
	eventKATimeOut        = "katimeout"
)

// =============================================================================

//go:embed duck-w.png
var logo []byte

func newClient() (agent.AgentClient, error) {
	return agent.NewClient("", "")
}

func wError(a fyne.App, err string) {
	beeep.Alert("Netmux: Error", err, "")
}

func wMsg(a fyne.App, msg string) {
	beeep.Alert("Netmux: Message", msg, "")
}

var winMain fyne.Window

var cli agent.AgentClient

func agCli() (agent.AgentClient, error) {
	var err error
	if cli == nil {
		cli, err = newClient()
	}
	return cli, err
}

func resetCli() {
	cli = nil
}

func Run(actuser *user.User) error {
	var err error
	a := app.New()

	if err != nil {
		wError(a, fmt.Sprintf("Error opening client: %s", err.Error()))
	}

	go func() {
		for {
			cli, err := agCli()
			if err != nil {
				resetCli()
				logrus.Warnf(err.Error())
				time.Sleep(time.Second)
				continue
			}
			streamEvt, err := cli.Events(context.Background(), &agent.Noop{})
			if err != nil {
				resetCli()
				logrus.Warnf(err.Error())
				time.Sleep(time.Second)
				continue
			}
			for {
				evt, err := streamEvt.Recv()
				if err != nil {
					resetCli()
					logrus.Warnf(err.Error())
					time.Sleep(time.Second)
					break
				}
				switch evt.MsgType {
				case eventTypeConnected:
					wMsg(a, fmt.Sprintf("Connected to: %s", evt.Ctx))
				case eventTypeDisconnected:
					wMsg(a, fmt.Sprintf("Disconnected from: %s", evt.Ctx))
				case eventKATimeOut:
				}
				if err != nil {
					wError(a, fmt.Sprintf("Error receiving event: %s", err.Error()))
					break
				}
			}
		}
	}()

	cli, err := agCli()
	if err != nil {
		resetCli()
		wError(a, err.Error())
		return err
	}

	_, err = cli.Load(context.Background(), &agent.StringMsg{Msg: filepath.Join(actuser.HomeDir, ".netmux.yaml")})

	if err != nil {
		wError(a, err.Error())
		return err
	}

	if desk, ok := a.(desktop.App); ok {
		mis := []*fyne.MenuItem{}

		mis = append(mis, fyne.NewMenuItem("Main Window", func() {
			ex, _ := os.Executable()
			cmd := exec.Command(ex, "webview")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				wError(a, err.Error())
			}

		}))

		mis = append(mis, fyne.NewMenuItem("Restart Agent", func() {
			cli, err := agCli()
			if err != nil {
				resetCli()
				wError(a, err.Error())
				return
			}
			cli.Exit(context.Background(), &agent.Noop{})
		}))

		menu := fyne.NewMenu("Netmux", mis...)

		desk.SetSystemTrayMenu(menu)

		desk.SetSystemTrayIcon(fyne.NewStaticResource("logo", logo))

		a.Lifecycle().SetOnStarted(func() {
			go func() {
				time.Sleep(200 * time.Millisecond)
				setActivationPolicy()
			}()
		})

		a.Run()
	}
	return nil
}
