package cmd

import (
	"os"
	"os/exec"
)

func sh(cmdline string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", cmdline)
	return cmd
}

func shStr(cmdline string) (string, error) {
	cmd := sh(cmdline)
	bs, err := cmd.CombinedOutput()
	return string(bs), err
}

func shStdio(cmdline string) error {
	cmd := sh(cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
