package commands

import (
	"os"

	"github.com/ardanlabs.com/netmux/app/cli/installer"
	"github.com/ardanlabs.com/netmux/foundation/shell"
)

func trayInstall() error {
	ret, err := installer.InstallTray()
	os.Stdout.WriteString(ret)
	return err
}

func trayUninstall() error {
	ret, err := shell.Launchctl.UninstallTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}

func trayEnable() error {
	ret, err := shell.Launchctl.EnableTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}

func trayDisable() error {
	ret, err := shell.Launchctl.DisableTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}
