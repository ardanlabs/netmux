package cli

import (
	"context"
	"os"
	"strings"

	"github.com/ardanlabs.com/netmux/lib/proto/agent"
)

func ctxOn() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.Connect(context.Background(), &agent.StringMsg{Msg: opts.Ctx.On.Ctx})
	return err
}

func ctxOff() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.Disconnect(context.Background(), &agent.StringMsg{Msg: opts.Ctx.Off.Ctx})
	return err
}
func ctxReset() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.Connect(context.Background(), &agent.StringMsg{Msg: opts.Ctx.Reset.Ctx})
	if err != nil {
		return err
	}
	_, err = cli.Disconnect(context.Background(), &agent.StringMsg{Msg: opts.Ctx.Reset.Ctx})
	return err
}

func ctxPing() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.Ping(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Ping.Ctx, Msg: strings.Join(opts.Ctx.Ping.Cmd, " ")})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(res.Msg + "\n")
	return err
}

func ctxPscan() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.PortScan(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Pscan.Ctx, Msg: strings.Join(opts.Ctx.Pscan.Cmd, " ")})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(res.Msg + "\n")
	return err
}

func ctxSpeedTest() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.SpeedTest(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Speedtest.Ctx, Msg: opts.Ctx.Speedtest.Pl})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(res.Msg + "\n")
	return err
}

func ctxNc() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.Nc(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Pscan.Ctx, Msg: strings.Join(opts.Ctx.Pscan.Cmd, " ")})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(res.Msg + "\n")
	return err
}

func ctxPfOn() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.PfOn(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Pfon.Ctx})
	return err
}
func ctxPfOff() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.PfOff(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Pfoff.Ctx})
	return err
}

func ctxInstall() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.ClusterInstall(context.Background(), &agent.ClusterInstallReq{
		Nxctx: opts.Ctx.Install.Ctx,
		Kctx:  opts.Ctx.Install.Kctx,
		Ns:    opts.Ctx.Install.Ns,
		Arch:  opts.Ctx.Install.Arch,
	})
	return err
}

func ctxUninstall() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.ClusterUninstall(context.Background(), &agent.StringMsg{Ctx: &opts.Ctx.Uninstall.Ctx})
	return err
}
