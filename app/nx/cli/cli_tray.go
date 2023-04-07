package cli

import (
	"os"

	"go.digitalcircle.com.br/dc/netmux/app/nx/installer"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
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
