package cli

import (
	"context"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/grpc"
	"go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
	"path/filepath"
	"strings"
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
