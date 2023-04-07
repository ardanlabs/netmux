package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/ardanlabs.com/netmux/app/nx/grpc"
	"github.com/ardanlabs.com/netmux/lib/proto/agent"
)

func server() error {

	return grpc.Run(actuser)

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
	_, err = cli.Load(context.Background(), &agent.StringMsg{Msg: opts.Load.Fname})
	return err
}
