package tray

import "C"
import (
	"context"
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/gen2brain/beeep"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/lib/events"
	"go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"
)

//go:embed duck-w.png
var logo []byte

func newClient() (agent.AgentClient, error) {
	return agent.NewUnixDefault()
}

func wError(err string) {
	errA := beeep.Alert("Netmux: Error", err, "")
	if errA != nil {
		logrus.Warnf("error on beep: %s", errA.Error())
	}
}

func wMsg(msg string) {
	errA := beeep.Alert("Netmux: Message", msg, "")
	if errA != nil {
		logrus.Warnf("error on beep: %s", errA.Error())
	}
}

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
		wError(fmt.Sprintf("Error opening client: %s", err.Error()))
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
				case events.EventTypeConnected:
					wMsg(fmt.Sprintf("Connected to: %s", evt.Ctx))
				case events.EventTypeDisconnected:
					wMsg(fmt.Sprintf("Disconnected from: %s", evt.Ctx))
				case events.EventKATimeOut:
				}
				if err != nil {
					wError(fmt.Sprintf("Error receiving event: %s", err.Error()))
					break
				}
			}
		}
	}()

	cli, err := agCli()
	if err != nil {
		resetCli()
		wError(err.Error())
		return err
	}

	_, err = cli.Load(context.Background(), &agent.StringMsg{Msg: filepath.Join(actuser.HomeDir, ".netmux.yaml")})

	if err != nil {
		wError(err.Error())
		return err
	}

	if desk, ok := a.(desktop.App); ok {
		var mis = make([]*fyne.MenuItem, 0)

		mis = append(mis, fyne.NewMenuItem("Main Window", func() {
			ex, _ := os.Executable()
			cmd := exec.Command(ex, "webview")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				wError(err.Error())
			}

		}))

		mis = append(mis, fyne.NewMenuItem("Restart Agent", func() {
			cli, err := agCli()
			if err != nil {
				resetCli()
				wError(err.Error())
				return
			}
			_, err = cli.Exit(context.Background(), &agent.Noop{})
			if err != nil {
				logrus.Warnf("error calling exit: %s", err.Error())
				return
			}
		}))

		var menu *fyne.Menu

		menu = fyne.NewMenu("Netmux", mis...)

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
