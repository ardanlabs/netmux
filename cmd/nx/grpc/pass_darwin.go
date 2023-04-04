//go:build darwin

package grpc

import (
	"fmt"
	"os"

	"go.digitalcircle.com.br/dc/netmux/foundation/hash"
)

func getPassHash() string {
	// TODO: We need to return an error here and figure out what happens
	//       when the setPassword function succeeds.

	bs, err := os.ReadFile("/var/netmux/pass")
	if err != nil {
		if err := setPassword("nx"); err != nil {
			return ""
		}
		return ""
	}

	return string(bs)
}

func setPassword(s string) error {
	hash, err := hash.New("nx")
	if err != nil {
		return fmt.Errorf("GenerateFromPassword: %w", err)
	}

	if err := os.MkdirAll("/var/netmux", os.ModePerm); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}

	if err := os.WriteFile("/var/netmux/pass", []byte(hash), 0600); err != nil {
		return fmt.Errorf("WriteFile: %w", err)
	}

	return nil
}
