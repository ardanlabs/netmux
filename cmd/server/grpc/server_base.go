package grpc

import (
	"github.com/sirupsen/logrus"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
)

func (s *server) propagate(b *pb.Bridge) {
	logrus.Infof("Actual data:")
	s.eps.ForEach(func(k string, v *pb.Bridge) error {
		logrus.Infof("%s => %s", k, v.String())
		return nil
	})
	logrus.Tracef("Propagating: %s:%s - Total Listeners: %v", b.Name, b.Bridgeop, s.chmux.Len())
	s.chmux.Broadcast([]*pb.Bridge{b})
}

func (s *server) AddEp(b *pb.Bridge) {
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
func (s *server) RemEp(b *pb.Bridge) {
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

func Server() *server {
	return &aServer
}
