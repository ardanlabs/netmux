package web

import (
	"context"
	"time"

	"github.com/ardanlabs.com/netmux/business/grpc/local"
	"github.com/sirupsen/logrus"
	"github.com/webview/webview"
)

type jsResponse struct {
	Data any   `json:"data"`
	Err  error `json:"err"`
}

var _cli local.LocalClient

func aCli() local.LocalClient {
	if _cli == nil {
		var err error
		_cli, err = local.NewClient("", "")
		if err != nil {
			_cli = nil
			logrus.Warnf(err.Error())
			time.Sleep(time.Second)
		}
	}
	return _cli
}
func resetCli() {
	_cli = nil
}
func Run() error {

	wv := webview.New(true)
	defer wv.Destroy()
	wv.SetTitle("Netmux GUI")
	//url := RunWebserverDev()
	url := RunWebserver()
	wv.Navigate(url)
	wv.SetSize(1200, 500, webview.HintNone)
	setActivationPolicy()

	wv.Bind("nx_start", func(a string) *jsResponse {
		res, err := aCli().Connect(context.Background(), &local.StringMsg{Msg: a})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}
	})

	wv.Bind("nx_stop", func(a string) *jsResponse {
		res, err := aCli().Disconnect(context.Background(), &local.StringMsg{Msg: a})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}
	})

	wv.Bind("nx_status", func() *jsResponse {
		res, err := aCli().Status(context.Background(), &local.StringMsg{})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Bind("nx_login", func(ctx string, user string, pass string) *jsResponse {
		res, err := aCli().Login(context.Background(), &local.LoginMessage{
			Username: user,
			Password: pass,
			Context:  ctx,
		})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Bind("nx_logout", func(ctx string) *jsResponse {
		res, err := aCli().Logout(context.Background(), &local.StringMsg{
			Msg:     ctx,
			Ctx:     &ctx,
			MsgType: nil,
		})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Bind("nx_svc_start", func(ctx string, svc string) *jsResponse {
		res, err := aCli().StartSvc(context.Background(), &local.SvcRequest{
			Ctx:  ctx,
			Svcs: []string{svc},
		})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Bind("nx_svc_stop", func(ctx string, svc string) *jsResponse {
		res, err := aCli().StopSvc(context.Background(), &local.SvcRequest{
			Ctx:  ctx,
			Svcs: []string{svc},
		})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Bind("nx_exit", func() *jsResponse {
		res, err := aCli().Exit(context.Background(), &local.Noop{})
		if err != nil {
			resetCli()
			return &jsResponse{Err: err}
		} else {
			return &jsResponse{Data: res}
		}

	})

	wv.Run()
	return nil
}
