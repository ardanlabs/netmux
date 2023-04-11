package cli

import (
	"context"
	"os"

	"go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
)

func ctxLogin() error {
	cli, err := newClient()

	if err != nil {
		return err
	}

	_, err = cli.Login(context.Background(), &agent.LoginMessage{
		Username: opts.User,
		Password: opts.Pass,
		Context:  opts.Ctx.Login.Ctx,
	})
	if err == nil {
		os.Stdout.WriteString("Logged IN w success")
	}
	if opts.Ctx.Login.On {
		opts.Ctx.On.Ctx = opts.Ctx.Login.Ctx
		err = ctxOn()
	}
	return err
}

func ctxLogout() error {
	cli, err := newClient()

	if err != nil {
		return err
	}

	_, err = cli.Logout(context.Background(), &agent.StringMsg{Msg: opts.Ctx.Logout.Ctx})
	if err == nil {
		os.Stdout.WriteString("Logged OUT w success")
	}
	return err
}
