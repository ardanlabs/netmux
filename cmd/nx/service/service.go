package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
	"go.digitalcircle.com.br/dc/netmux/lib/hosts"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"go.digitalcircle.com.br/dc/netmux/lib/types"
)

type Service struct {
	parent *Context
	Bridge *types.Bridge
	ctx    context.Context
	cancel func()
	// Transient info

	Sent         int64        `yaml:"-"`
	Recv         int64        `yaml:"-"`
	NConns       int          `yaml:"-"`
	Status       Status       `yaml:"-"`
	IpAddr       string       `yaml:"-"`
	Listener     net.Listener `yaml:"-" json:"-"`
	uuidHostname string       `yaml:"-"`
	uuidIfconfig string       `yaml:"-" `
}

func (s *Service) Prepare(ctx *Context) {
	s.parent = ctx

}

func (s *Service) Name() string {
	return s.Bridge.Name
}

func (s *Service) JsonBytes() []byte {
	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(s)
	return buf.Bytes()
}

func (s *Service) ResetCounters() {

	s.Sent = 0
	s.Recv = 0

}

func (s *Service) listen() error {
	s.Status = StatusConnecting
	var err error

	s.IpAddr = ipMgr.Allocate()

	logrus.Debugf("Listening service %s: %s", s.Name(), s.IpAddr)

	err = shell.Ifconfig.AddAlias(Default().Iface, s.IpAddr)

	if err != nil {
		return err
	}

	s.uuidIfconfig = TermHanlder.add(func() error {
		s.uuidIfconfig = ""
		err = shell.Ifconfig.RemoveAlias(Default().Iface, s.IpAddr)
		if err != nil {
			logrus.Warnf("error reseting alias: %v", err)
		}
		return nil
	})
	def := hosts.Default()
	def.Add(s.IpAddr, []string{s.Bridge.LocalAddr}, fmt.Sprintf("#nx: ctx:(%s) ep:(%s)", s.parent.Name, s.Name()))

	s.uuidHostname = TermHanlder.add(func() error {
		s.uuidHostname = ""
		def.RemoveByComment(fmt.Sprintf("ep:(%s)", s.Name()))

		if err != nil {
			logrus.Warnf("error reseting alias: %v", err)
		}
		return nil
	})

	if err != nil {

		return err
	}
	s.Bridge.LocalAddr = s.IpAddr
	logrus.Tracef("Agent will listen: %s", s.Bridge.String())
	s.Listener, err = s.Bridge.ListenerLocal()

	if err != nil {
		logrus.Warnf("error listening: %v", err)
		return err
	}
	s.Status = StatusAvailable

	defer func() {
		s.Status = StatusDisconnected
	}()

	go func() {
		<-s.ctx.Done()
		if s.Listener != nil {
			s.Listener.Close()
		}
	}()

	for {
		c, err := s.Listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnGrpc(c)

	}
}

func (s *Service) StartForward() error {

	go func() {
		err := s.listen()
		if err != nil {
			logrus.Warnf("service.Start::error listening: %s", err.Error())
		}
	}()

	if s.Listener != nil {
		_ = s.Listener.Close()
	}
	return nil
}

func (s *Service) Start() error {

	if s.Status == StatusStarted {
		return fmt.Errorf("service already started")
	}
	s.ctx, s.cancel = context.WithCancel(s.parent.ctx)
	logrus.Tracef("Setting up service: %s", s.Name())
	s.Status = StatusStarted
	var err error
	defer func() {
		if err != nil {
			s.Status = StatusError
		}
	}()

	switch s.Bridge.Direction {
	case types.BridgeReward:
		chErr := make(chan error)
		go s.StartReverseServiceGrpc(chErr)
		if err := <-chErr; err != nil {
			logrus.Warnf("error setting up ref conn for %s: %s", s.Name(), err.Error())
		}
		close(chErr)

	case types.BridgeForward:

		err = s.StartForward()
		if err != nil {
			logrus.Warnf("error setting up ref conn for %s: %s", s.Name(), err.Error())
			return err
		}

	default:
		err = fmt.Errorf("Direction %s is unknown for service %s", s.Bridge.Direction, s.Name())
	}

	return err
}

func (s *Service) Stop() error {
	if s.Status == StatusStopped {
		return fmt.Errorf("service already stopped")
	}
	s.Status = StatusStopping
	if s.cancel != nil {
		s.cancel()
	}
	if s.Listener != nil {
		logrus.Debugf("Closing listener for: %s", s.Name())
		err := s.Listener.Close()
		if err != nil {
			logrus.Warnf("Error closing listener for %s: %s", s.Name(), err.Error())
		}
	}
	TermHanlder.TerminateSome(s.uuidHostname, s.uuidIfconfig)
	s.Status = StatusStopped
	return nil
}

func (s *Service) handleConnGrpc(c net.Conn) error {
	cli := s.parent.cli

	proxyStream, err := cli.Proxy(context.Background())
	if err != nil {
		return err
	}

	pbb := &pb.Bridge{}
	s.Bridge.ToPb(pbb)
	err = proxyStream.Send(&pb.ConnOut{
		Bridge: pbb,
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
			s.Sent = s.Sent + int64(n)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}

			err = proxyStream.Send(&pb.ConnOut{
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
			s.Recv = s.Recv + int64(len(ci.Pl))

			_, err = c.Write(ci.Pl)
			if err != nil {
				c.Close()
				chErr <- err
				return
			}
		}
	}()

	s.NConns++
	s.parent.NConns++
	rawErr := <-chErr
	s.NConns--
	s.parent.NConns--

	logrus.Infof("Closing con %s => %s", s.Name(), s.Bridge.String())

	if err != nil {
		err = rawErr.(error)
		logrus.Warnf("error client conn: %v", err)
	}

	return err
}

func (s *Service) StartReverseServiceGrpc(chErr chan error) {
	cli := s.parent.cli

	lcli, err := cli.ReverseProxyListen(s.ctx)

	if err != nil {
		logrus.Warnf("Error setting up rev proxy listener conn %s: %s", s.Name(), err.Error())
		chErr <- err
		return
	}

	pbb := &pb.Bridge{}
	s.Bridge.ToPb(pbb)
	err = lcli.Send(&pb.ConnOut{Bridge: pbb})
	if err != nil {
		logrus.Warnf("Error opening remote rev proxy listener conn %s: %s", s.Name(), err.Error())
		chErr <- err
		return
	}

	go func() {
		<-s.ctx.Done()
		logrus.Warnf("ctx for %s cancelled - closing revproxy conn.", s.Name())
		s.cancel()
	}()
	chErr <- nil
	for {
		req, err := lcli.Recv()
		if err != nil {
			logrus.Warnf("Error receiving new reverse conn from listener %s: %s", s.Name(), err.Error())
			s.cancel()
			return
		}
		logrus.Debugf("Got new rev proxy request: %s", req.ConnId)
		go s.handleDataConnGrpc(req.ConnId)
	}

}

func (s *Service) handleDataConnGrpc(id string) {
	cli := s.parent.cli
	rpw, err := cli.ReverseProxyWork(context.Background())
	if err != nil {
		logrus.Warnf("error opening ReverseProxyWork: %s", err.Error())
		return
	}
	c, err := s.Bridge.DialLocal()
	if err != nil {
		logrus.Warnf("error dialing: %s", err.Error())
		return
	}

	err = rpw.Send(&pb.RevProxyConnIn{
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
		err = rpw.Send(&pb.RevProxyConnIn{
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
