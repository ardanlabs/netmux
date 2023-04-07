package auth

import (
	"log"
	"testing"
)

func TestLogin(t *testing.T) {
	ret, err := Login("nx", "nx")
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Print(ret)
}
