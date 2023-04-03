package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hpcloud/tail"
	"github.com/rodaine/table"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/cli/monitor"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/cli/tray"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/cli/webview"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/installer"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/service"
	"go.digitalcircle.com.br/dc/netmux/foundation/argon2"
	"go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
)

func newClient() (agent.AgentClient, error) {
	return agent.NewUnixDefault()
}

func start() error {
	return nil
}

func stop() error {
	return nil
}

func printOut(s string, p ...any) {
	_, _ = os.Stdout.WriteString(fmt.Sprintf(s+"\n", p...))
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
		err = exit()
	case "server":
		err = server()

	case "webview":
		err = webview.Run()
	case "status":
		err = status()
	case "tray install":
		err = trayInstall()
	case "tray uninstall":
		err = trayUninstall()
	case "tray enable":
		err = trayEnable()
	case "tray disable":
		err = trayDisable()
	case "tray run":
		err = tray.Run(actuser)
	case "ctx install <ctx> <kctx> <ns> <arch>":
		err = ctxInstall()
	case "ctx uninstall <ctx>":
		err = ctxUninstall()
	case "ctx login <ctx>":
		err = ctxLogin()
	case "ctx logout <ctx>":
		err = ctxLogout()
	case "ctx on <ctx>":
		err = ctxOn()
	case "ctx off <ctx>":
		err = ctxOff()
	case "ctx pfon <ctx>":
		err = ctxPfOn()
	case "ctx pfoff <ctx>":
		err = ctxPfOff()
	case "ctx reset <ctx>":
		err = ctxReset()
	case "ctx ping <ctx> <cmd>":
		err = ctxPing()
	case "ctx pscan <ctx> <cmd>":
		err = ctxPscan()
	case "ctx speedtest <ctx> <pl>":
		err = ctxSpeedTest()
	case "ctx nc <ctx> <cmd>":
		err = ctxNc()
	case "svc on <ctx> <svc>":
		err = svcOn()
	case "svc off <ctx> <svc>":
		err = svcOff()
	case "svc reset <ctx> <svc>":
		err = svcReset()

	case "config set <fname>":
		err = configSet()
	case "config dump":
		err = configDump()
	case "config get":
		err = configGet()
	case "config hosts show":
		err = hostsShow()
	case "config hosts reset":
		err = hostsReset()

	case "version":
		logrus.Infof("%s: %s", service.Default().Semver, service.Default().Version)

	case "monitor":
		err = monitorHandler()

	case "logs":
		t, err := tail.TailFile("/tmp/netmux.log", tail.Config{Follow: true})
		if err != nil {
			return err
		}
		for line := range t.Lines {
			printOut(line.Text)
		}
	case "agent install":
		err := installer.Install()
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
		err = epList()

	case "auth hash":
		gen, err := argon2.GenerateHash(opts.Pass)
		if err != nil {
			return err
		}
		println(gen)
	default:
		err = fmt.Errorf("unknown command: %s", cliCtx.Command())
	}
	return err
}
