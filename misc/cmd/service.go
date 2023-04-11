package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"os"
	user2 "os/user"
	"time"
)

func handleCli(c net.Conn) error {
	for {
		msg := fmt.Sprintf("%s: User: %s, Orig: %s\n", time.Now().String(), username, origName)
		_, err := c.Write([]byte(msg))
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

var origName string
var username string

func main() {
	user, err := user2.Current()
	if err != nil {
		panic(err)
	}
	username = user.Username
	if username == "root" {
		origName = os.Getenv("SUDO_USER")
	}
	_ = os.RemoveAll("/tmp/service.sock")
	l, err := net.Listen("unix", "/tmp/service.sock")
	if err != nil {
		panic(err)
	}
	err = os.Chmod("/tmp/service.sock", 0777)
	if err != nil {
		panic(err)
	}
	for {
		cli, err := l.Accept()
		if err != nil {
			log.Printf("Error on accept: %s", err.Error())
		}
		go func() {
			err := handleCli(cli)
			if err != nil {
				logrus.Warnf("main.handleCli::error writing to server: %s", err.Error())
			}
		}()
	}
}
