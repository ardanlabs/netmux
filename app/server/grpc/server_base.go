package grpc

import (
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/sirupsen/logrus"
)

func (s *Server) propagate(brd *proxy.Bridge) {
	logrus.Infof("Actual data:")

	s.eps.ForEach(func(k string, brd *proxy.Bridge) error {
		logrus.Infof("%s => %s", k, brd.String())
		return nil
	})

	logrus.Tracef("Propagating: %s:%s", brd.Name, brd.Bridgeop)
	s.signal.Broadcast(brd)
}

func (s *Server) AddProxyBridge(b *proxy.Bridge) {
	b.Bridgeop = "A"
	existing, _ := s.eps.Get(b.Name)
	if existing != nil {
		if existing.K8Snamespace == b.K8Snamespace && existing.K8Sname == b.K8Sname && existing.K8Skind == b.K8Skind {
			logrus.Infof("Adding ep: %s", b.String())
			s.eps.Set(b.Name, b)
			s.propagate(b)
		}
		return
	}
	s.eps.Set(b.Name, b)
	s.propagate(b)
}
func (s *Server) RemEp(b *proxy.Bridge) {
	b.Bridgeop = "D"
	existing, _ := s.eps.Get(b.Name)
	if existing != nil {
		if existing.K8Snamespace == b.K8Snamespace && existing.K8Sname == b.K8Sname && existing.K8Skind == b.K8Skind {
			logrus.Infof("Removing ep: %s", b.String())
			s.eps.Delete(b.Name)
			s.propagate(b)
		}
		return
	}
	s.propagate(b)
}
