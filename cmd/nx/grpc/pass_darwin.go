//go:build darwin

package grpc

import (
	"go.digitalcircle.com.br/dc/netmux/lib/argon2"
	"os"
)

func getPassHash() string {
	bs, err := os.ReadFile("/var/netmux/pass")
	hash := ""
	if err != nil {
		err = setPassword("nx")
	} else {
		hash = string(bs)
	}
	return hash
}

func setPassword(s string) error {
	hash, _ := argon2.GenerateFromPassword("nx")
	os.MkdirAll("/var/netmux", os.ModePerm)
	return os.WriteFile("/var/netmux/pass", []byte(hash), 0600)
}
