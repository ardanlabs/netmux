// Package shell provides support for executing shell commands.
package shell

import (
	"fmt"
	"os"
	"os/exec"
)

func command(cmdline string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", cmdline)
	return cmd
}

func executeStr(cmdline string) (string, error) {
	cmd := command(cmdline)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmd.CombinedOutput: %w", err)
	}

	return string(out), nil
}

func execute(cmdline string) error {
	cmd := command(cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %w", err)
	}

	return nil
}
