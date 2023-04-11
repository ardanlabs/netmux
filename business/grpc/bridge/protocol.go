package bridge

import "fmt"

// Set of available protocols.
var (
	ProtoTCP  = Protocol{"tcp"}
	ProtoTCP4 = Protocol{"tcp4"}
	ProtoTCP6 = Protocol{"tcp6"}
)

// Set of known protocols.
var protocols = map[string]Protocol{
	ProtoTCP.name:  ProtoTCP,
	ProtoTCP4.name: ProtoTCP4,
	ProtoTCP4.name: ProtoTCP4,
}

// =============================================================================

// Protocol defines a bridge protocol.
type Protocol struct {
	name string
}

// ParseProtocol parses the string value and returns a protocol if one exists.
func ParseProtocol(value string) (Protocol, error) {
	protocol, exists := protocols[value]
	if !exists {
		return Protocol{}, fmt.Errorf("invalid protocol: %q", value)
	}

	return protocol, nil
}

// MustParseProtocol parses the string value and returns a protocol if one
// exists. If an error occurs the function panics.
func MustParseProtocol(value string) Direction {
	protocol, err := ParseDirection(value)
	if err != nil {
		panic(err)
	}

	return protocol
}

// Name returns the name of the status.
func (p Protocol) Name() string {
	return p.name
}

// UnmarshalText implement the unmarshal interface for JSON conversions.
func (p *Protocol) UnmarshalText(data []byte) error {
	p.name = string(data)
	return nil
}

// MarshalText implement the marshal interface for JSON conversions.
func (p Protocol) MarshalText() ([]byte, error) {
	return []byte(p.name), nil
}

// Equal provides support for the go-cmp package and testing.
func (p Protocol) Equal(p2 Protocol) bool {
	return p.name == p2.name
}
