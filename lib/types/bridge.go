package types

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"gopkg.in/yaml.v3"
	"net"
)

const (
	BridgeForward = "F"
	BridgeReward  = "R"
)

type Bridge struct {
	Name       string `yaml:"name"`
	LocalAddr  string `yaml:"localAddr"`
	LocalPort  string `yaml:"localPort"`
	RemoteAddr string `yaml:"remoteAddr"`
	RemotePort string `yaml:"remotePort"`
	Proto      string `yaml:"proto"`
	Direction  string `yaml:"direction"`
	Auto       bool   `yaml:"auto"`
}

func (b *Bridge) String() string {
	return fmt.Sprintf("%#v", b)
}

func (b *Bridge) LocalAddrStr() string {
	return fmt.Sprintf("%s:%s", b.LocalAddr, b.LocalPort)
}

func (b *Bridge) RemoteAddrStr() string {
	return fmt.Sprintf("%s:%s", b.RemoteAddr, b.RemotePort)
}

func (b *Bridge) ListenerLocal() (net.Listener, error) {

	logrus.Tracef("Bridge: trying to listen to: %s: %s", b.Proto, b.LocalAddrStr())
	ret, err := net.Listen(b.Proto, b.LocalAddrStr())
	if err != nil {
		logrus.Warnf("Error creating local listener: %s", err.Error())
	}
	return ret, err
}

func (b *Bridge) ListenerRemote() (net.Listener, error) {
	logrus.Tracef("Bridge: trying to listen to: %s: %s", b.Proto, b.RemoteAddrStr())
	ret, err := net.Listen(b.Proto, b.RemoteAddrStr())
	if err != nil {
		logrus.Warnf("Error creating remote listener: %s", err.Error())
	}
	return ret, err
}

func (b *Bridge) ListenerOnRemoteHost() (net.Listener, error) {
	logrus.Tracef("Bridge: trying on localhost to listen to: %s: %s", b.Proto, b.RemotePort)
	ret, err := net.Listen(b.Proto, ":"+b.RemotePort)
	if err != nil {
		logrus.Warnf("Error creating remote listener: %s", err.Error())
	}
	return ret, err
}

func (b *Bridge) DialLocal() (net.Conn, error) {
	return net.Dial(b.Proto, b.LocalAddrStr())
}

func (b *Bridge) DialRemote() (net.Conn, error) {
	return net.Dial(b.Proto, b.RemoteAddrStr())
}

func (b *Bridge) FromPb(bridge *pb.Bridge) {
	b.RemotePort = bridge.Remoteport
	b.RemoteAddr = bridge.Remoteaddr
	b.LocalPort = bridge.Localport
	b.LocalAddr = bridge.Localaddr
	b.Proto = bridge.Proto
	b.Name = bridge.Name
	b.Direction = bridge.Direction
	b.Auto = bridge.Auto
}

func (b *Bridge) ToPb(dst *pb.Bridge) {
	dst.Name = b.Name
	dst.Remoteport = b.RemotePort
	dst.Remoteaddr = b.RemoteAddr
	dst.Localport = b.LocalPort
	dst.Localaddr = b.LocalAddr
	dst.Proto = b.Proto
	dst.Direction = b.Direction
	dst.Auto = b.Auto
}

func (e *Bridge) LoadFromAnnotation(s string) error {
	err := yaml.Unmarshal([]byte(s), e)
	if err != nil {
		return err
	}
	if e.Name == "" {
		e.Name = uuid.NewString()
	}
	return nil
}

func NewBridge() *Bridge {
	ret := Bridge{Name: uuid.NewString(), Direction: "F", Proto: "tcp"}
	return &ret
}

type Bridges []*Bridge

func (b *Bridges) LoadFromAnnotation(s string) error {
	err := yaml.Unmarshal([]byte(s), b)
	if err != nil {
		return err
	}
	for _, e := range *b {
		if e.Direction == "" {
			e.Direction = "F"
		}
		if e.Proto == "" {
			e.Proto = "tcp"
		}
	}
	return nil
}

func NewBridges() Bridges {
	ret := make([]*Bridge, 0)

	return ret
}
