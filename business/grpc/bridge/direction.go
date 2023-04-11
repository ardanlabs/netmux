package bridge

import "fmt"

// Set of available directions.
var (
	DirectionForward = Direction{"F"}
	DirectionReward  = Direction{"R"}
)

// Set of known directions.
var directions = map[string]Direction{
	DirectionForward.name: DirectionForward,
	DirectionReward.name:  DirectionReward,
}

// =============================================================================

// Direction defines a bridge direction.
type Direction struct {
	name string
}

// ParseDirection parses the string value and returns a direction if one exists.
func ParseDirection(value string) (Direction, error) {
	direction, exists := directions[value]
	if !exists {
		return Direction{}, fmt.Errorf("invalid direction: %q", value)
	}

	return direction, nil
}

// MustParseDirection parses the string value and returns a direction if one
// exists. If an error occurs the function panics.
func MustParseDirection(value string) Direction {
	direction, err := ParseDirection(value)
	if err != nil {
		panic(err)
	}

	return direction
}

// Name returns the name of the status.
func (d Direction) Name() string {
	return d.name
}

// UnmarshalText implement the unmarshal interface for JSON conversions.
func (d *Direction) UnmarshalText(data []byte) error {
	d.name = string(data)
	return nil
}

// MarshalText implement the marshal interface for JSON conversions.
func (d Direction) MarshalText() ([]byte, error) {
	return []byte(d.name), nil
}

// Equal provides support for the go-cmp package and testing.
func (d Direction) Equal(d2 Direction) bool {
	return d.name == d2.name
}
