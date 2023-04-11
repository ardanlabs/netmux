package cluster

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
	"github.com/ardanlabs.com/netmux/foundation/hosts"
	"github.com/ardanlabs.com/netmux/foundation/shell"
	"github.com/sirupsen/logrus"
)

type Status string
type RuntimeType string

const (
	StatusDisconnected Status      = "disconnected"
	StatusConnecting   Status      = "connecting"
	StatusAvailable    Status      = "available"
	StatusDisabled     Status      = "disabled"
	StatusStarting     Status      = "starting"
	StatusStarted      Status      = "started"
	StatusRunning      Status      = "running"
	StatusStopping     Status      = "stopping"
	StatusStopped      Status      = "stopped"
	StatusLoading      Status      = "loading"
	StatusError        Status      = "error"
	RuntimeKubernetes  RuntimeType = "kubernetes"

	Sock = "/tmp/netmux.sock"
)

// stats maintains stats for accessing the cluster service.
type stats struct {
	status      Status
	sent        int64
	recv        int64
	connections int
}

// Cluster provides communication access to the cluster service.
type Cluster struct {
	parent       *Context
	bridge       bridge.Bridge
	cancel       func()
	hosts        *hosts.Hosts
	listener     net.Listener
	ipAddr       string
	uuidHostname string
	uuidIfconfig string
	stats        stats
	shutdown     context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
}

// Start starts the communication between the local service and cluster service.
func Start(direction bridge.Direction) (*Cluster, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cluster := Cluster{
		shutdown: cancel,
	}

	cluster.wg.Add(1)

	go func() {
		defer cluster.wg.Done()

		switch direction {
		case bridge.DirectionReward:
			cluster.StartReverseServiceGrpc(ctx)

		case bridge.DirectionForward:
			cluster.StartForward(ctx)
		}
	}()

	return &cluster, nil
}

// Shutdown signals all the running G's to shutdown and waits.
func (c *Cluster) Shutdown() {
	c.shutdown()
	c.wg.Wait()
}

// =============================================================================

// StartListening
func (c *Cluster) StartListening() error {
	go func() {
		err := s.listen()
		if err != nil {
			logrus.Warnf("service.Start::error listening: %s", err.Error())
		}
	}()

	if s.listener != nil {
		_ = s.listener.Close()
	}
	return nil
}

func (c *Cluster) listen() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.status = StatusConnecting
	var err error

	c.ipAddr = ipMgr.Allocate()

	logrus.Debugf("Listening service %s: %s", c.bridge.Name, c.ipAddr)

	err = shell.Ifconfig.AddAlias(Default().Iface, c.ipAddr)

	if err != nil {
		return err
	}

	c.uuidIfconfig = TermHanlder.add(func() error {
		c.uuidIfconfig = ""
		err = shell.Ifconfig.RemoveAlias(Default().Iface, c.ipAddr)
		if err != nil {
			logrus.Warnf("error reseting alias: %v", err)
		}
		return nil
	})
	c.hosts.Add(c.ipAddr, []string{c.bridge.LocalHost}, fmt.Sprintf("#nx: ctx:(%s) ep:(%s)", c.parent.Name, c.bridge.Name))

	c.uuidHostname = TermHanlder.add(func() error {
		c.uuidHostname = ""
		c.hosts.Remove(fmt.Sprintf("ep:(%s)", c.bridge.Name))

		if err != nil {
			logrus.Warnf("error reseting alias: %v", err)
		}
		return nil
	})

	if err != nil {

		return err
	}
	c.bridge.LocalHost = c.ipAddr
	logrus.Tracef("Agent will listen: %s", c.bridge.String())
	c.listener, err = c.bridge.LocalListener()
	if err != nil {
		logrus.Warnf("error listening: %v", err)
		return err
	}
	c.stats.status = StatusAvailable

	defer func() {
		c.stats.status = StatusDisconnected
	}()

	go func() {
		// <-c.ctx.Done()
		if c.listener != nil {
			c.listener.Close()
		}
	}()

	for {
		conn, err := c.listener.Accept()
		if err != nil {
			return err
		}
		go c.handleConnGrpc(conn)

	}
}

func (s *Cluster) Stop() error {
	if s.stats.status == StatusStopped {
		return fmt.Errorf("service already stopped")
	}
	s.stats.status = StatusStopping
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		logrus.Debugf("Closing listener for: %s", s.bridge.Name)
		err := s.listener.Close()
		if err != nil {
			logrus.Warnf("Error closing listener for %s: %s", s.bridge.Name, err.Error())
		}
	}
	TermHanlder.TerminateSome(s.uuidHostname, s.uuidIfconfig)
	s.stats.status = StatusStopped
	return nil
}

func (s *Cluster) handleConnGrpc(c net.Conn) error {
	cli := s.parent.cli

	proxyStream, err := cli.Proxy(context.Background())
	if err != nil {
		return err
	}

	err = proxyStream.Send(&cluster.ConnOut{
		Bridge: bridge.NewClusterBridge(s.bridge),
		Pl:     nil,
	})
	if err != nil {
		return err
	}
	chErr := make(chan error)

	//bridge := s.Bridge
	//fout, _ := os.OpenFile(fmt.Sprintf("%s_%s_%v.out", bridge.RemoteAddr, bridge.RemotePort, time.Now().UnixMilli()), os.O_CREATE|os.O_RDWR, 0600)
	//fin, _ := os.OpenFile(fmt.Sprintf("%s_%s_%v.in", bridge.RemoteAddr, bridge.RemotePort, time.Now().UnixMilli()), os.O_CREATE|os.O_RDWR, 0600)

	//defer fout.Close()
	//defer fin.Close()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.Read(buf)
			s.stats.sent = s.stats.sent + int64(n)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}

			err = proxyStream.Send(&cluster.ConnOut{
				Pl: buf[:n],
			})
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
		}
	}()

	go func() {
		for {
			ci, err := proxyStream.Recv()
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
			s.stats.recv = s.stats.recv + int64(len(ci.Pl))

			_, err = c.Write(ci.Pl)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
		}
	}()

	s.stats.connections++
	s.parent.NConns++
	<-chErr
	s.stats.connections--
	s.parent.NConns--

	logrus.Infof("Closing con %s => %s", s.bridge.Name, s.bridge)

	if err != nil {
		logrus.Warnf("error client conn: %v", err)
	}

	return err
}

func (s *Cluster) StartReverseServiceGrpc(chErr chan error) {
	cli := s.parent.cli

	lcli, err := cli.ReverseProxyListen(s.ctx)

	if err != nil {
		logrus.Warnf("Error setting up rev proxy listener conn %s: %s", s.bridge.Name, err.Error())
		chErr <- err
		return
	}

	err = lcli.Send(&cluster.ConnOut{Bridge: bridge.NewClusterBridge(s.bridge)})
	if err != nil {
		logrus.Warnf("Error opening remote rev proxy listener conn %s: %s", s.bridge.Name, err.Error())
		chErr <- err
		return
	}

	go func() {
		<-s.ctx.Done()
		logrus.Warnf("ctx for %s cancelled - closing revproxy conn.", s.bridge.Name)
		s.cancel()
	}()
	chErr <- nil
	for {
		req, err := lcli.Recv()
		if err != nil {
			logrus.Warnf("Error receiving new reverse conn from listener %s: %s", s.bridge.Name, err.Error())
			s.cancel()
			return
		}
		logrus.Debugf("Got new rev proxy request: %s", req.ConnId)
		go s.handleDataConnGrpc(req.ConnId)
	}

}

func (s *Cluster) handleDataConnGrpc(id string) {
	cli := s.parent.cli
	rpw, err := cli.ReverseProxyWork(context.Background())
	if err != nil {
		logrus.Warnf("error opening ReverseProxyWork: %s", err.Error())
		return
	}
	c, err := s.bridge.LocalDial()
	if err != nil {
		logrus.Warnf("error dialing: %s", err.Error())
		return
	}

	err = rpw.Send(&cluster.RevProxyConnIn{
		ConnId: id,
		Pl:     nil,
	})
	if err != nil {
		logrus.Warnf("error connecting: %s", err.Error())
		return
	}

	chErr := make(chan error)

	go func() {
		buf := make([]byte, 4096)
		n, err := c.Read(buf)
		if err != nil {
			chErr <- err
			c.Close()
			return
		}
		err = rpw.Send(&cluster.RevProxyConnIn{
			ConnId: id,
			Pl:     buf[:n],
		})
		if err != nil {
			chErr <- err
			c.Close()
			return
		}
	}()

	go func() {
		res, err := rpw.Recv()
		if err != nil {
			chErr <- err
			c.Close()
			return
		}
		_, err = c.Write(res.Pl)
		if err != nil {
			chErr <- err
			c.Close()
			return
		}
	}()

	err = <-chErr
	if err != nil {
		logrus.Warnf("error processing: %s", err.Error())
		return
	}
}
