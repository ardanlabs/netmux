// Package grpc provides support for SOMETHING!!
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/service"
	"go.digitalcircle.com.br/dc/netmux/lib/config"
	"go.digitalcircle.com.br/dc/netmux/lib/events"
	"go.digitalcircle.com.br/dc/netmux/lib/hosts"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/agent"
	pbs "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

type server struct {
	pb.UnsafeAgentServer
}

func (s server) Load(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	_, err := os.Stat(req.Msg)
	if err != nil {
		return nil, err
	}
	service.Default().Load(req.Msg)
	return &pb.Noop{}, nil
}

func (s server) SetConfig(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	logrus.Infof("Loading config from: %s", req.Msg)
	service.Reset()
	config.Default().Fname = req.Msg
	err := config.Default().Save()
	if err != nil {
		return nil, err
	}
	err = service.Default().Load(req.Msg)
	return &pb.Noop{}, err
}

func (s server) GetConfig(ctx context.Context, req *pb.Noop) (*pb.StringMsg, error) {
	res := &pb.StringMsg{Msg: config.Default().Fname}
	return res, nil

}

func (s server) Connect(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)

	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}

	nxctx.Token = config.Default().Tokens[req.Msg]

	var chCtxReady = make(chan struct{})
	var chErr = make(chan error)

	go func() {
		err := nxctx.Start(context.Background(), chCtxReady)
		if err != nil {
			select {
			case chErr <- err:
			default:
				logrus.Warnf("error from ctx getconfig: %s", err.Error())
			}
		}
	}()

	select {
	case <-chCtxReady:
	case err := <-chErr:
		return nil, err
	case <-time.After(time.Second * 30):
		return nil, fmt.Errorf("timeout connecting to context: %s", req.Msg)
	}

	ka, err := nxctx.Cli().KeepAlive(context.Background(), &pbs.Noop{})
	if err != nil {
		return nil, err
	}

	go func() {
		var t = time.NewTicker(time.Second * 5)
		var nErr = 0
		go func() {
			for nErr < 5 {
				<-t.C
				events.Send(&events.Event{
					Type:    events.EventKATimeOut,
					Ctx:     nxctx.Name,
					Payload: "Timeout for keepalive",
				})
				nErr++
			}
		}()

		for {
			_, err = ka.Recv()
			if err != nil {
				nErr++
				events.Send(&events.Event{
					Type:    events.EventTypeDisconnected,
					Ctx:     nxctx.Name,
					Payload: fmt.Sprintf("Disconnected: %s", err.Error()),
				})
				nErr = 6
				return
			}
			nErr = 0
			t.Reset(time.Second * 5)
		}
	}()

	events.Send(&events.Event{
		Type:    events.EventTypeConnected,
		Ctx:     req.Msg,
		Payload: "connected at: " + time.Now().String(),
	})

	return &pb.Noop{}, err
}

func (s server) Disconnect(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	nxctx.Stop()
	nxctx.Services = make([]*service.Service, 0)

	return &pb.Noop{}, nil
}

func (s server) ClusterInstall(ctx context.Context, req *pb.ClusterInstallReq) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Nxctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := service.DefaultK8sController().SetupDeploy(ctx, nxctx, req.Ns, req.Kctx, req.Arch)
	return &pb.Noop{}, err
}

func (s server) ClusterUninstall(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := service.DefaultK8sController().TearDownDeploy(ctx, nxctx)
	return &pb.Noop{}, err
}

func (s server) PfOn(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StartPortForwarding()
	return &pb.Noop{}, err
}

func (s server) PfOff(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StopPortForwarding()
	return &pb.Noop{}, err
}

func (s server) Login(ctx context.Context, req *pb.LoginMessage) (*pb.StringMsg, error) {

	nxctx := service.Default().CtxByName(req.Context)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Context)
	}
	tk, err := nxctx.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	config.Default().Tokens[req.Context] = tk
	err = config.Default().Save()
	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: "ok"}, nil
}

func (s server) Logout(ctx context.Context, req *pb.StringMsg) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Msg)
	}
	err := nxctx.Logout()
	if err != nil {
		return nil, err
	}
	config.Default().Tokens[req.Msg] = ""
	err = config.Default().Save()
	if err != nil {
		return nil, err
	}

	return &pb.Noop{}, nil
}

func (s server) ResetHosts(ctx context.Context, req *pb.Noop) (*pb.Noop, error) {
	hosts.Default().RemoveByComment("nx: ctx")
	return &pb.Noop{}, nil
}

func (s server) StartSvc(ctx context.Context, req *pb.SvcRequest) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("ctx %s not found", req.Ctx)
	}

	for i := range req.Svcs {
		svcn := req.Svcs[i]
		svc := nxctx.SvcByName(svcn)
		if svc == nil {
			return nil, fmt.Errorf("svc %s.%s not found", req.Ctx, svcn)
		}
		var err error
		go func() {
			err = svc.Start()
			if err != nil {
				logrus.Warnf("Error starting service: %s", err.Error())
			}
		}()
		time.Sleep(time.Second)
	}

	return &pb.Noop{}, nil
}

func (s server) StopSvc(ctx context.Context, req *pb.SvcRequest) (*pb.Noop, error) {
	nxctx := service.Default().CtxByName(req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("ctx %s not found", req.Ctx)
	}
	for i := range req.Svcs {
		svcn := req.Svcs[i]
		svc := nxctx.SvcByName(svcn)
		if svc == nil {
			return nil, fmt.Errorf("svc %s.%s not found", req.Ctx, svcn)
		}
		err := svc.Stop()
		if err != nil {
			logrus.Warnf("Error stopping service: %s", err.Error())
		}
	}
	return &pb.Noop{}, nil
}

func (s server) Exit(ctx context.Context, req *pb.Noop) (*pb.Noop, error) {
	os.Exit(0)
	return nil, nil
}

func (s server) Monitor(req *pb.Noop, res pb.Agent_MonitorServer) error {
	for {
		bs, _ := json.Marshal(service.Default())
		err := res.Send(&pb.StringMsg{Msg: string(bs)})
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

func (s server) Events(req *pb.Noop, res pb.Agent_EventsServer) error {
	l := events.NewListener()

	defer func() {
		events.Close(l)
	}()

	for {
		evt := <-l
		res.Send(&pb.EventMsg{
			Msg:     evt.PayloadJson(),
			Ctx:     evt.Ctx,
			MsgType: string(evt.Type),
		})

	}
}

func buildStatus(resetCounters bool) *pb.StatusResponse {
	var ret = pb.StatusResponse{}
	ret.Fname = config.Default().Fname
	ret.Version = service.Ver + " - " + service.Semver
	for i := range service.Default().Contexts {
		actx := service.Default().Contexts[i]
		ctx := &pb.ContextStatusResponse{}
		ctx.Name = actx.Name
		ctx.Status = string(actx.Status)
		ctx.Pfstatus = string(actx.PortForwardStatus)
		ctx.Portforward = fmt.Sprintf("%#v", actx.EnablePortForward)
		for j := range actx.Services {
			asvc := actx.Services[j]
			if asvc.Bridge != nil {
				svc := &pb.ServiceStatusResponse{
					Name:       asvc.Name(),
					Localaddr:  asvc.Bridge.LocalAddr,
					Localport:  asvc.Bridge.LocalPort,
					Remoteaddr: asvc.Bridge.RemoteAddr,
					Remoteport: asvc.Bridge.RemotePort,
					Proto:      asvc.Bridge.Proto,
					Auto:       fmt.Sprintf("%#v", asvc.Bridge.Auto),
					Status:     string(asvc.Status),
					Dir:        asvc.Bridge.Direction,
					Nconns:     int32(asvc.NConns),
					Sent:       asvc.Sent,
					Recv:       asvc.Recv,
				}
				ctx.Services = append(ctx.Services, svc)
			}

		}
		ret.Contexts = append(ret.Contexts, ctx)
	}
	if resetCounters {
		service.Default().ResetCounters()
	}
	return &ret
}

func (s server) Status(_ context.Context, req *pb.StringMsg) (*pb.StatusResponse, error) {
	reset := req.Msg == "zero"
	res := buildStatus(reset)
	return res, nil

}
func (s server) Ping(ctx context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := service.Default().CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().Ping(ctx, &pbs.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: res.Msg}, nil
}
func (s server) PortScan(ctx context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := service.Default().CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().PortScan(ctx, &pbs.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: res.Msg}, nil
}
func (s server) SpeedTest(ctx context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	start := time.Now()
	res, err := nxctx.Cli().SpeedTest(ctx, &pbs.StringMsg{
		Msg: req.Msg,
	})
	dur := time.Since(start)
	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: fmt.Sprintf("Got payload from ctx %s. Total time: %s. Pl Size: %s / %v",
		*req.Ctx,
		dur.String(),
		req.String(),
		len(res.Msg),
	)}, nil
}
func (s server) Config(context.Context, *pb.Noop) (*pb.BytesMsg, error) {
	bs, err := json.Marshal(service.Default())
	if err != nil {
		return nil, err
	}
	ret := &pb.BytesMsg{Msg: bs}
	return ret, nil
}

func (s server) Nc(ctx context.Context, req *pb.StringMsg) (*pb.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := service.Default().CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().Nc(ctx, &pbs.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &pb.StringMsg{Msg: res.Msg}, nil
}

func Run(actuser *user.User) error {

	lblog := &lumberjack.Logger{
		Filename:   "/tmp/netmux.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	}
	log.SetOutput(lblog)
	logrus.SetOutput(lblog)

	logrus.Infof("Starting server: %s / %s", service.Default().Semver, service.Default().Semver)

	logrus.Infof("Running server as: %s", actuser.Username)
	cfg := os.Getenv("CONFIG")
	if cfg != "" {
		logrus.Infof("Using config from env: %s", cfg)
		config.Default().Src = cfg
	} else {
		config.Default().Src = "/etc/netmux.yaml"
	}
	err := config.Default().Load()
	if err != nil {
		return err
	}
	if config.Default().Fname != "" {
		logrus.Infof("Using config: %s", config.Default().Fname)
		count := 0
		for {
			_, err := os.Stat(config.Default().Fname)
			if err == nil {
				logrus.Infof("Loading config: %s", config.Default().Fname)
				service.Default().Load(config.Default().Fname)

				break
			}
			if err != nil {
				logrus.Warnf("couldnt load config - will give it a new try")
				count++
				time.Sleep(time.Second)
				continue
			}
			if count == 30 {
				logrus.Warnf("couldnt load config for 30 times - aborting")
				panic("couldnt load config")
			}
		}

	}

	_ = os.RemoveAll(service.Sock)
	l, err := net.Listen("unix", service.Sock)
	if err != nil {
		return err
	}
	err = os.Chmod(service.Sock, 0777)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAgentServer(grpcServer, server{})
	return grpcServer.Serve(l)
}
