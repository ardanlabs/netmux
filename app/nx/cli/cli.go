package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/ardanlabs.com/netmux/app/nx/cli/monitor"
	"github.com/ardanlabs.com/netmux/app/nx/cli/tray"
	"github.com/ardanlabs.com/netmux/app/nx/cli/webview"
	"github.com/ardanlabs.com/netmux/app/nx/installer"
	"github.com/ardanlabs.com/netmux/app/nx/service"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
	"github.com/ardanlabs.com/netmux/foundation/hash"
	"github.com/hpcloud/tail"
	"github.com/rodaine/table"
	"github.com/sirupsen/logrus"
)

func newClient() (agent.AgentClient, error) {
	return agent.New(opts.User, opts.Pass)
}

func start() error {
	return nil
}

func stop() error {
	return nil
}

func exit() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.Exit(context.Background(), &agent.Noop{})
	return err
}

func monitorHandler() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	return monitor.Run(cli, map[string]func() error{"start": start, "stop": stop})
}

func epList() error {
	ag, err := newClient()
	if err != nil {
		return err
	}
	res, err := ag.Config(context.Background(), &agent.Noop{})
	if err != nil {
		return err
	}

	cfg := &service.Netmux{}
	err = json.Unmarshal(res.Msg, cfg)
	if err != nil {
		return err
	}
	table.DefaultHeaderFormatter = func(format string, vals ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	}

	tbl := table.New("ID", "Ctx", "Name", "Direction", "Enabled", "Connected")
	for ctxi, ctx := range cfg.Contexts {
		var ctxEnabled = "N"
		var ctxConnected = "N"

		if ctx.Enabled {
			ctxEnabled = "Y"
		}
		if ctx.Cli() != nil {
			ctxConnected = "Y"
		}

		for svci, svc := range ctx.Services {

			tbl.AddRow(fmt.Sprintf("%v.%v", ctxi, svci), ctx.Name, svc.Name, svc.Bridge.Direction, ctxEnabled, ctxConnected)
		}
	}

	tbl.Print()
	return nil
}

var actuser *user.User

func Run() error {
	var err error
	actuser, err = user.Current()
	if err != nil {
		return err
	}

	cliCtx := kong.Parse(&opts)

	switch cliCtx.Command() {
	case "exit":
		exit()
	case "server":
		server()

	case "webview":
		return webview.Run()
	case "status":
		return status()
	case "tray install":
		return trayInstall()
	case "tray uninstall":
		return trayUninstall()
	case "tray enable":
		return trayEnable()
	case "tray disable":
		return trayDisable()
	case "tray run":
		return tray.Run(actuser)
	case "ctx install <ctx> <kctx> <ns> <arch>":
		return ctxInstall()
	case "ctx uninstall <ctx>":
		return ctxUninstall()
	case "ctx login <ctx>":
		return ctxLogin()
	case "ctx logout <ctx>":
		return ctxLogout()
	case "ctx on <ctx>":
		return ctxOn()
	case "ctx off <ctx>":
		return ctxOff()
	case "ctx pfon <ctx>":
		return ctxPfOn()
	case "ctx pfoff <ctx>":
		return ctxPfOff()
	case "ctx reset <ctx>":
		return ctxReset()
	case "ctx ping <ctx> <cmd>":
		return ctxPing()
	case "ctx pscan <ctx> <cmd>":
		return ctxPscan()
	case "ctx speedtest <ctx> <pl>":
		return ctxSpeedTest()
	case "ctx nc <ctx> <cmd>":
		return ctxNc()
	case "svc on <ctx> <svc>":
		return svcOn()
	case "svc off <ctx> <svc>":
		return svcOff()
	case "svc reset <ctx> <svc>":
		return svcReset()

	case "config set <fname>":
		return configSet()
	case "config dump":
		return configDump()
	case "config get":
		return configGet()
	case "config hosts show":
		return hostsShow()
	case "config hosts reset":
		return hostsReset()

	case "version":
		logrus.Infof("%s: %s", service.Default().Semver, service.Default().Version)
		return nil

	case "monitor":
		return monitorHandler()

	case "logs":
		t, err := tail.TailFile("/tmp/netmux.log", tail.Config{Follow: true})
		if err != nil {
			panic(err)
		}
		for line := range t.Lines {
			_, _ = os.Stdout.WriteString(line.Text + "\n")
		}
	case "agent install":
		err := installer.Install(opts.Agent.Install.Ctx, opts.Agent.Install.Ns)
		if err != nil {
			return err
		}
	case "agent autoinstall":
		err := installer.AutoInstall(opts.Agent.Autoinstall.Ctx, opts.Agent.Autoinstall.Ns, opts.Agent.Autoinstall.Arch)
		if err != nil {
			return err
		}
	case "agent uninstall":
		err := installer.Uninstall()
		if err != nil {
			return err
		}

	case "ep ls":
		return epList()

	case "auth hash":
		gen, err := hash.New(opts.Pass)
		if err != nil {
			return err
		}
		println(gen)
	default:
		return fmt.Errorf("unknown command: %s", cliCtx.Command())
	}
	return nil
}
