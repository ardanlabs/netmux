// Package grpc provides support for SOMETHING!!
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"sync"
	"time"

	server "github.com/ardanlabs.com/netmux/app/services/local/cluster"
	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
	"github.com/ardanlabs.com/netmux/business/grpc/local"
	"github.com/ardanlabs.com/netmux/business/sys/nmconfig"
	"github.com/ardanlabs.com/netmux/foundation/hosts"
	"github.com/ardanlabs.com/netmux/foundation/signal"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// GRPC implements the grpc interface.
type GRPC struct {
	local.UnsafeLocalServer
	log     *logrus.Logger
	user    *user.User
	cluster *server.Cluster
	local   *grpc.Server
	signal  *signal.Signal[event]
	hosts   *hosts.Hosts
	wg      sync.WaitGroup
}

// Start starts the grpc server.
func Start(log *logrus.Logger, user *user.User, cluster *server.Cluster) (*GRPC, error) {
	grpc := GRPC{
		log:     log,
		user:    user,
		cluster: cluster,
		local:   grpc.NewServer(),
		signal:  signal.New[event](),
		hosts:   hosts.New(),
	}

	os.RemoveAll(server.Sock)
	l, err := net.Listen("unix", server.Sock)
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}

	if err := os.Chmod(server.Sock, 0777); err != nil {
		return nil, fmt.Errorf("os.Chmod: %w", err)
	}

	local.RegisterLocalServer(grpc.local, &grpc)

	grpc.wg.Add(1)

	go func() {
		log.Info("grpc: started")
		defer func() {
			log.Info("grpc: shutdown")
			grpc.wg.Done()
		}()

		grpc.log.Infof("grpc: running server at 0.0.0.0:48080")
		grpc.local.Serve(l)
	}()

	return &grpc, nil
}

// Shutdown requests the proxy service to stop and waits.
func (g *GRPC) Shutdown() {
	g.log.Infof("grpc: starting shutdown")
	defer g.log.Infof("grpc: shutdown")

	g.local.GracefulStop()
}

// =============================================================================

/*
  rpc SetConfig(StringMsg) returns(Noop){}
  rpc GetConfig(Noop) returns(StringMsg){}

  rpc Connect(StringMsg) returns (Noop){}
  rpc Disconnect(StringMsg) returns (Noop){}
  rpc PfOn(StringMsg) returns (Noop){}
  rpc PfOff(StringMsg) returns (Noop){}
  rpc StartSvc(SvcRequest) returns (Noop){}
  rpc StopSvc(SvcRequest) returns (Noop){}
  rpc ResetHosts(Noop) returns(Noop){}
  rpc SpeedTest(StringMsg) returns (StringMsg){}

  rpc Load(StringMsg) returns(Noop){}
  rpc Exit(Noop) returns(Noop){}

  rpc ClusterInstall(ClusterInstallReq) returns (Noop){}
  rpc ClusterUninstall(StringMsg) returns (Noop){}

  rpc Login(LoginMessage) returns(StringMsg){}
  rpc Logout(StringMsg) returns(Noop){}

  rpc Config(Noop) returns(BytesMsg){}
  rpc Status(StringMsg) returns(StatusResponse){}
  rpc Monitor(Noop) returns(stream StringMsg){}
  rpc Events(Noop)  returns(stream EventMsg){}

  rpc Ping(StringMsg) returns(StringMsg){}
  rpc PortScan(StringMsg) returns(StringMsg){}
  rpc Nc(StringMsg) returns(StringMsg){}
*/

func (g *GRPC) SetConfig(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	g.log.Infof("Loading config from: %s", req.Msg)
	g.cluster.Reset()

	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	if err := cfg.UpdateFileName(req.Msg); err != nil {
		return nil, err
	}

	if err := g.cluster.Load(req.Msg); err != nil {
		return nil, err
	}

	return &local.Noop{}, err
}

func (g *GRPC) Load(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	_, err := os.Stat(req.Msg)
	if err != nil {
		return nil, err
	}
	g.cluster.Load(req.Msg)
	return &local.Noop{}, nil
}

func (g *GRPC) GetConfig(ctx context.Context, req *local.Noop) (*local.StringMsg, error) {
	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	res := &local.StringMsg{Msg: cfg.FileName()}

	return res, nil
}

func (g *GRPC) Connect(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Msg)

	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}

	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	nxctx.Token, _ = cfg.Token(req.Msg)

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

	ka, err := nxctx.Cli().KeepAlive(context.Background(), &cluster.Noop{})
	if err != nil {
		return nil, err
	}

	go func() {
		var t = time.NewTicker(time.Second * 5)
		var nErr = 0
		go func() {
			for nErr < 5 {
				<-t.C
				g.signal.Broadcast(event{
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
				g.signal.Broadcast(event{
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

	g.signal.Broadcast(event{
		Type:    eventTypeConnected,
		Ctx:     req.Msg,
		Payload: "connected at: " + time.Now().String(),
	})

	return &local.Noop{}, err
}

func (g *GRPC) Disconnect(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	nxctx.Stop()
	nxctx.Services = make([]*server.Cluster, 0)

	return &local.Noop{}, nil
}

func (g *GRPC) ClusterInstall(ctx context.Context, req *local.ClusterInstallReq) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Nxctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := g.cluster.DefaultK8sController().SetupDeploy(ctx, nxctx, req.Ns, req.Kctx, req.Arch)
	return &local.Noop{}, err
}

func (g *GRPC) ClusterUninstall(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	err := g.cluster.DefaultK8sController().TearDownDeploy(ctx, nxctx)
	return &local.Noop{}, err
}

func (g *GRPC) PfOn(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StartPortForwarding()
	return &local.Noop{}, err
}

func (g *GRPC) PfOff(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(*req.Ctx)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found")
	}
	if !nxctx.EnablePortForward {
		return nil, fmt.Errorf("context has no port forward")
	}
	err := nxctx.StopPortForwarding()
	return &local.Noop{}, err
}

func (g *GRPC) Login(ctx context.Context, req *local.LoginMessage) (*local.StringMsg, error) {
	nxctx := g.cluster.CtxByName(req.Context)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Context)
	}

	tk, err := nxctx.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	if err := cfg.AddToken(req.Context, tk); err != nil {
		return nil, err
	}

	return &local.StringMsg{Msg: "ok"}, nil
}

func (g *GRPC) Logout(ctx context.Context, req *local.StringMsg) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Msg)
	if nxctx == nil {
		return nil, fmt.Errorf("context not found: %s", req.Msg)
	}

	if err := nxctx.Logout(); err != nil {
		return nil, err
	}

	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	if err := cfg.DeleteToken(req.Msg); err != nil {
		return nil, err
	}

	return &local.Noop{}, nil
}

func (g *GRPC) ResetHosts(ctx context.Context, req *local.Noop) (*local.Noop, error) {
	g.hosts.Remove("nx: ctx")
	return &local.Noop{}, nil
}

func (g *GRPC) StartSvc(ctx context.Context, req *local.SvcRequest) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Ctx)
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

	return &local.Noop{}, nil
}

func (g *GRPC) StopSvc(ctx context.Context, req *local.SvcRequest) (*local.Noop, error) {
	nxctx := g.cluster.CtxByName(req.Ctx)
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
	return &local.Noop{}, nil
}

func (g *GRPC) Exit(ctx context.Context, req *local.Noop) (*local.Noop, error) {
	os.Exit(0)
	return nil, nil
}

func (g *GRPC) Monitor(req *local.Noop, res local.Local_MonitorServer) error {
	for {
		bs, _ := json.Marshal(g.cluster)
		err := res.Send(&local.StringMsg{Msg: string(bs)})
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

func (g *GRPC) Events(req *local.Noop, res local.Local_EventsServer) error {
	chn := g.signal.Aquire()

	for {
		evt, ok := <-chn
		if !ok {
			return nil
		}

		res.Send(&local.EventMsg{
			Msg:     evt.PayloadJson(),
			Ctx:     evt.Ctx,
			MsgType: string(evt.Type),
		})
	}
}

func (g *GRPC) buildStatus(resetCounters bool) (*local.StatusResponse, error) {
	var ret = local.StatusResponse{}

	cfg, err := nmconfig.Load()
	if err != nil {
		return nil, err
	}

	ret.Fname = cfg.FileName()
	// ret.Version = g.cluster.Ver + " - " + g.cluster.Semver

	for i := range g.cluster.Contexts {
		actx := g.cluster.Contexts[i]
		ctx := &local.ContextStatusResponse{}
		ctx.Name = actx.Name
		ctx.Status = string(actx.Status)
		ctx.Pfstatus = string(actx.PortForwardStatus)
		ctx.Portforward = fmt.Sprintf("%#v", actx.EnablePortForward)
		for j := range actx.Services {
			asvc := actx.Services[j]
			if asvc.Bridge.IsZero() {
				svc := &local.ServiceStatusResponse{
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
		g.cluster.ResetCounters()
	}

	return &ret, nil
}

func (g *GRPC) Status(_ context.Context, req *local.StringMsg) (*local.StatusResponse, error) {
	res, err := g.buildStatus(req.Msg == "zero")
	if err != nil {
		return nil, err
	}

	return res, nil

}
func (g *GRPC) Ping(ctx context.Context, req *local.StringMsg) (*local.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := g.cluster.CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().Ping(ctx, &cluster.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &local.StringMsg{Msg: res.Msg}, nil
}
func (g *GRPC) PortScan(ctx context.Context, req *local.StringMsg) (*local.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := g.cluster.CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().PortScan(ctx, &cluster.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &local.StringMsg{Msg: res.Msg}, nil
}
func (g *GRPC) SpeedTest(ctx context.Context, req *local.StringMsg) (*local.StringMsg, error) {
	nxctx := g.cluster.CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	start := time.Now()
	res, err := nxctx.Cli().SpeedTest(ctx, &cluster.StringMsg{
		Msg: req.Msg,
	})
	dur := time.Since(start)
	if err != nil {
		return nil, err
	}

	return &local.StringMsg{Msg: fmt.Sprintf("Got payload from ctx %s. Total time: %s. Pl Size: %s / %v",
		*req.Ctx,
		dur.String(),
		req.String(),
		len(res.Msg),
	)}, nil
}
func (g *GRPC) Config(context.Context, *local.Noop) (*local.BytesMsg, error) {
	bs, err := json.Marshal(g.cluster)
	if err != nil {
		return nil, err
	}
	ret := &local.BytesMsg{Msg: bs}
	return ret, nil
}

func (g *GRPC) Nc(ctx context.Context, req *local.StringMsg) (*local.StringMsg, error) {
	if req.Ctx == nil {
		return nil, fmt.Errorf("no ctx provided")
	}

	nxctx := g.cluster.CtxByName(*req.Ctx)

	if nxctx == nil {
		return nil, fmt.Errorf("context %s not found", ctx)
	}
	if nxctx.Cli() == nil {
		err := fmt.Errorf("context %s not connected to server", nxctx.Name)
		return nil, err
	}
	res, err := nxctx.Cli().Nc(ctx, &cluster.StringMsg{
		Msg: req.Msg,
	})

	if err != nil {
		return nil, err
	}

	return &local.StringMsg{Msg: res.Msg}, nil
}
