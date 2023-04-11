package bridge

import (
	"fmt"
	"net"

	"github.com/ardanlabs.com/netmux/business/grpc/cluster"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Set of bridge types.
const (
	DirectionForward = "F"
	DirectionReward  = "R"
)

// Set of protocols.
const (
	ProtoTCP  = "tcp"
	ProtoTCP4 = "tcp4"
	ProtoTCP6 = "tcp6"
)

// =============================================================================

// Bridges represetns a collection of bridge values.
type Bridges []Bridge

// LoadBridges constructs a collection of bridges based on the specified
// annotation.
func LoadBridges(annotation string) (Bridges, error) {
	var bs Bridges
	if err := yaml.Unmarshal([]byte(annotation), &bs); err != nil {
		return nil, err
	}

	for _, b := range bs {
		if b.Direction == "" {
			b.Direction = DirectionForward
		}

		if b.Proto == "" {
			b.Proto = ProtoTCP
		}
	}

	return bs, nil
}

// =============================================================================

// Bridge represents a networking bridge.
type Bridge struct {
	Name       string `yaml:"name"`
	LocalHost  string `yaml:"localHost"`
	LocalPort  string `yaml:"localPort"`
	RemoteHost string `yaml:"remoteHost"`
	RemotePort string `yaml:"remotePort"`
	Proto      string `yaml:"proto"`
	Direction  string `yaml:"direction"`
	Auto       bool   `yaml:"auto"`
}

// New converts the specified cluster bridge value and marshals it
// into a bridge value.
func New(proxy *cluster.Bridge) Bridge {
	return Bridge{
		LocalPort:  proxy.Localport,
		LocalHost:  proxy.Localaddr,
		RemotePort: proxy.Remoteport,
		RemoteHost: proxy.Remoteaddr,
		Proto:      proxy.Proto,
		Name:       proxy.Name,
		Direction:  proxy.Direction,
		Auto:       proxy.Auto,
	}
}

// NewClusterBridge converts the specified bridge value and marshals it into
// a protocol buffers bridge value.
func NewClusterBridge(b Bridge) *cluster.Bridge {
	return &cluster.Bridge{
		Localport:  b.LocalPort,
		Localaddr:  b.LocalHost,
		Remoteport: b.RemotePort,
		Remoteaddr: b.RemoteHost,
		Proto:      b.Proto,
		Name:       b.Name,
		Direction:  b.Direction,
		Auto:       b.Auto,
	}
}

// Load converts the specified annotation into a Bridge.
func Load(annotation string) (Bridge, error) {
	var b Bridge
	if err := yaml.Unmarshal([]byte(annotation), &b); err != nil {
		return Bridge{}, err
	}

	if b.Name == "" {
		b.Name = uuid.NewString()
	}

	return b, nil
}

// String implements the stringer interface.
func (b Bridge) String() string {
	return fmt.Sprintf("%#v", b)
}

// IsZero checks if the bridge value is empty.
func (b Bridge) IsZero() bool {
	return b == Bridge{}
}

// LocalListener announces on the configured local network.
func (b Bridge) LocalListener() (net.Listener, error) {
	lsn, err := net.Listen(b.Proto, b.localHostPort())
	if err != nil {
		return nil, fmt.Errorf("local listener: %w", err)
	}

	return lsn, err
}

// RemoteListener announces on the configured remote network.
func (b Bridge) RemoteListener() (net.Listener, error) {
	lsn, err := net.Listen(b.Proto, b.remoteHostPort())
	if err != nil {
		return nil, fmt.Errorf("remote listener: %w", err)
	}

	return lsn, err
}

// RemotePortListener announces on the configured remote network port.
func (b Bridge) RemotePortListener() (net.Listener, error) {
	lsn, err := net.Listen(b.Proto, ":"+b.RemotePort)
	if err != nil {
		return nil, fmt.Errorf("remote port listener: %w", err)
	}

	return lsn, err
}

// LocalDial connects to the configured local network.
func (b Bridge) LocalDial() (net.Conn, error) {
	return net.Dial(b.Proto, b.localHostPort())
}

// RemoteDial connects to the configured remote network.
func (b Bridge) RemoteDial() (net.Conn, error) {
	return net.Dial(b.Proto, b.remoteHostPort())
}

// =============================================================================

func (b Bridge) localHostPort() string {
	return net.JoinHostPort(b.LocalHost, b.LocalPort)
}

func (b Bridge) remoteHostPort() string {
	return net.JoinHostPort(b.RemoteHost, b.RemotePort)
}
