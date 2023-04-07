package cli

import (
	"context"

	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
)

func svcOn() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.StartSvc(context.Background(), &agent.SvcRequest{Ctx: opts.Svc.On.Ctx, Svcs: opts.Svc.On.Svc})
	return err
}

func svcOff() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.StopSvc(context.Background(), &agent.SvcRequest{Ctx: opts.Svc.Off.Ctx, Svcs: opts.Svc.Off.Svc})
	return err
}
func svcReset() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	_, err = cli.StartSvc(context.Background(), &agent.SvcRequest{Ctx: opts.Svc.Reset.Ctx, Svcs: opts.Svc.Reset.Svc})
	if err != nil {
		return err
	}
	_, err = cli.StopSvc(context.Background(), &agent.SvcRequest{Ctx: opts.Svc.Reset.Ctx, Svcs: opts.Svc.Reset.Svc})
	return err
}
