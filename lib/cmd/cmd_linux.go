package cmd

import "fmt"

func Ping(h string) (string, error) {
	return shStr(fmt.Sprintf("ping -c 4 %s", h))
}

func Nmap(h string) (string, error) {
	return shStr(fmt.Sprintf("nmap %s", h))
}

func Nc(h string) (string, error) {
	return shStr(fmt.Sprintf("nc %s", h))
}
