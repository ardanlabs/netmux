package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
)

func configSet() error {
	fname := opts.Config.Set.Fname
	if strings.HasPrefix(fname, "~") {
		strings.Replace(fname, "~", actuser.HomeDir, 1)
	}
	var err error
	fname, err = filepath.Abs(fname)
	if err != nil {
		return err
	}
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.SetConfig(context.Background(), &agent.StringMsg{Msg: fname})
	return err
}

func configGet() error {
	fname := opts.Config.Set.Fname
	if strings.HasPrefix(fname, "~") {
		strings.Replace(fname, "~", actuser.HomeDir, 1)
	}
	var err error
	if _, err = filepath.Abs(fname); err != nil {
		return err
	}
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.GetConfig(context.Background(), &agent.Noop{})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(res.Msg)
	return err
}

func configDump() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	res, err := cli.Config(context.Background(), &agent.Noop{})
	if err != nil {
		return err
	}
	os.Stdout.Write(res.Msg)
	return nil
}

func hostsShow() error {
	bs, _ := os.ReadFile("/etc/hosts")
	os.Stdout.Write(bs)
	os.Stdout.WriteString("\n")
	return nil
}

func hostsReset() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.ResetHosts(context.Background(), &agent.Noop{})
	if err != nil {
		return err
	}
	return nil
}
