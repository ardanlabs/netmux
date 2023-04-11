package commands

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/ardanlabs.com/netmux/business/grpc/local"
)

func server() error {

	//return grpc.Start(actuser)
	return nil

}

func load() error {
	if opts.Load.Fname == "" {
		opts.Load.Fname = filepath.Join(actuser.HomeDir, ".netmux.yaml")
	}
	opts.Load.Fname = strings.Replace(opts.Load.Fname, "~", actuser.HomeDir, 1)
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.Load(context.Background(), &local.StringMsg{Msg: opts.Load.Fname})
	return err
}
