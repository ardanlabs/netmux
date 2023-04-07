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

	"github.com/ardanlabs.com/netmux/app/nx/service"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/agent"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/config"
	"github.com/ardanlabs.com/netmux/foundation/hosts"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

const (
	eventTypeDisconnected = "disconnected"
	eventTypeConnected    = "connected"
	eventKATimeOut        = "katimeout"
)

type event struct {
	Type    string
	Ctx     string
	Payload any
}

func (e *event) PayloadJson() []byte {
	bs, _ := json.Marshal(e.Payload)
	return bs
}

// =============================================================================

type server struct {
	agent.UnsafeAgentServer
	signal *signal.Signal[event]
	hosts  *hosts.Hosts
}

func (s *server) Load(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	_, err := os.Stat(req.Msg)
	if err != nil {
		return nil, err
	}
	service.Default().Load(req.Msg)
	return &agent.Noop{}, nil
}

func (s *server) SetConfig(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	logrus.Infof("Loading config from: %s", req.Msg)
	service.Reset()

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if _, err := cfg.UpdateFName(req.Msg); err != nil {
		return nil, err
	}

	if err := service.Default().Load(req.Msg); err != nil {
		return nil, err
	}

	return &agent.Noop{}, err
}

func (s *server) GetConfig(ctx context.Context, req *agent.Noop) (*agent.StringMsg, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	res := &agent.StringMsg{Msg: cfg.FName}

	return res, nil
}

func (s *server) Connect(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)

	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	nxctx.Token = cfg.Tokens[req.Msg]

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

	ka, err := nxctx.Cli().KeepAlive(context.Background(), &proxy.Noop{})
	if err != nil {
		return nil, err
	}

	go func() {
		var t = time.NewTicker(time.Second * 5)
		var nErr = 0
		go func() {
			for nErr < 5 {
				<-t.C
				s.signal.Broadcast(event{
					Type:    eventKATimeOut,
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
				s.signal.Broadcast(event{
					Type:    eventTypeDisconnected,
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

	s.signal.Broadcast(event{
		Type:    eventTypeConnected,
		Ctx:     req.Msg,
		Payload: "connected at: " + time.Now().String(),
	})

	return &agent.Noop{}, err
}

func (s *server) Disconnect(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	nxctx.Stop()
	nxctx.Services = make([]*service.Service, 0)

	return &agent.Noop{}, nil
}

func (s *server) ClusterInstall(ctx context.Context, req *agent.ClusterInstallReq) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(req.Nxctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := service.DefaultK8sController().SetupDeploy(ctx, nxctx, req.Ns, req.Kctx, req.Arch)
	return &agent.Noop{}, err
}

func (s *server) ClusterUninstall(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := service.DefaultK8sController().TearDownDeploy(ctx, nxctx)
	return &agent.Noop{}, err
}

func (s *server) PfOn(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StartPortForwarding()
	return &agent.Noop{}, err
}

func (s *server) PfOff(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StopPortForwarding()
	return &agent.Noop{}, err
}

func (s *server) Login(ctx context.Context, req *agent.LoginMessage) (*agent.StringMsg, error) {
	nxctx := service.Default().CtxByName(req.Context)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Context)
	}

	tk, err := nxctx.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	cfg.Tokens[req.Context] = tk
	if err := cfg.Save(); err != nil {
		return nil, err
	}

	return &agent.StringMsg{Msg: "ok"}, nil
}

func (s *server) Logout(ctx context.Context, req *agent.StringMsg) (*agent.Noop, error) {
	nxctx := service.Default().CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Msg)
	}

	if err := nxctx.Logout(); err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	cfg.Tokens[req.Msg] = ""
	if err := cfg.Save(); err != nil {
		return nil, err
	}

	return &agent.Noop{}, nil
}

func (s *server) ResetHosts(ctx context.Context, req *agent.Noop) (*agent.Noop, error) {
	s.hosts.Remove("nx: ctx")
	return &agent.Noop{}, nil
}

func (s *server) StartSvc(ctx context.Context, req *agent.SvcRequest) (*agent.Noop, error) {
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

	return &agent.Noop{}, nil
}

func (s *server) StopSvc(ctx context.Context, req *agent.SvcRequest) (*agent.Noop, error) {
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
	return &agent.Noop{}, nil
}

func (s *server) Exit(ctx context.Context, req *agent.Noop) (*agent.Noop, error) {
	os.Exit(0)
	return nil, nil
}

func (s *server) Monitor(req *agent.Noop, res agent.Agent_MonitorServer) error {
	for {
		bs, _ := json.Marshal(service.Default())
		err := res.Send(&agent.StringMsg{Msg: string(bs)})
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

func (s *server) Events(req *agent.Noop, res agent.Agent_EventsServer) error {
	chn := s.signal.Aquire()

	for {
		evt, ok := <-chn
		if !ok {
			return nil
		}

		res.Send(&agent.EventMsg{
			Msg:     evt.PayloadJson(),
			Ctx:     evt.Ctx,
			MsgType: string(evt.Type),
		})
	}
}

func buildStatus(resetCounters bool) (*agent.StatusResponse, error) {
	var ret = agent.StatusResponse{}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	ret.Fname = cfg.FName
	ret.Version = service.Ver + " - " + service.Semver

	for i := range service.Default().Contexts {
		actx := service.Default().Contexts[i]
		ctx := &agent.ContextStatusResponse{}
		ctx.Name = actx.Name
		ctx.Status = string(actx.Status)
		ctx.Pfstatus = string(actx.PortForwardStatus)
		ctx.Portforward = fmt.Sprintf("%#v", actx.EnablePortForward)
		for j := range actx.Services {
			asvc := actx.Services[j]
			if asvc.Bridge.IsZero() {
				svc := &agent.ServiceStatusResponse{
					Name:       asvc.Name(),
					Localaddr:  asvc.Bridge.LocalHost,
					Localport:  asvc.Bridge.LocalPort,
					Remoteaddr: asvc.Bridge.RemoteHost,
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

	return &ret, nil
}

func (s *server) Status(_ context.Context, req *agent.StringMsg) (*agent.StatusResponse, error) {
	res, err := buildStatus(req.Msg == "zero")
	if err != nil {
		return nil, err
	}

	return res, nil

}
func (s *server) Ping(ctx context.Context, req *agent.StringMsg) (*agent.StringMsg, error) {
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
	res, err := nxctx.Cli().Ping(ctx, &proxy.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &agent.StringMsg{Msg: res.Msg}, nil
}
func (s *server) PortScan(ctx context.Context, req *agent.StringMsg) (*agent.StringMsg, error) {
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
	res, err := nxctx.Cli().PortScan(ctx, &proxy.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &agent.StringMsg{Msg: res.Msg}, nil
}
func (s *server) SpeedTest(ctx context.Context, req *agent.StringMsg) (*agent.StringMsg, error) {
	nxctx := service.Default().CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	start := time.Now()
	res, err := nxctx.Cli().SpeedTest(ctx, &proxy.StringMsg{
		Msg: req.Msg,
	})
	dur := time.Since(start)
	if err != nil {
		return nil, err
	}

	return &agent.StringMsg{Msg: fmt.Sprintf("Got payload from ctx %s. Total time: %s. Pl Size: %s / %v",
		*req.Ctx,
		dur.String(),
		req.String(),
		len(res.Msg),
	)}, nil
}
func (s *server) Config(context.Context, *agent.Noop) (*agent.BytesMsg, error) {
	bs, err := json.Marshal(service.Default())
	if err != nil {
		return nil, err
	}
	ret := &agent.BytesMsg{Msg: bs}
	return ret, nil
}

func (s *server) Nc(ctx context.Context, req *agent.StringMsg) (*agent.StringMsg, error) {
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
	res, err := nxctx.Cli().Nc(ctx, &proxy.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &agent.StringMsg{Msg: res.Msg}, nil
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
	cfgSource := os.Getenv("CONFIG")
	if cfgSource == "" {
		cfgSource = "/etc/netmux.yaml"
	}

	cfg, err := config.LoadFile(cfgSource)
	if err != nil {
		return err
	}

	if cfg.FName != "" {
		logrus.Infof("Using config: %s", cfg.FName)
		count := 0
		for {
			_, err := os.Stat(cfg.FName)
			if err == nil {
				logrus.Infof("Loading config: %s", cfg.FName)
				service.Default().Load(cfg.FName)

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

	srv := server{
		signal: signal.New[event](),
		hosts:  hosts.New(),
	}
	defer srv.signal.Shutdown()

	grpcServer := grpc.NewServer()
	agent.RegisterAgentServer(grpcServer, &srv)

	return grpcServer.Serve(l)
}
