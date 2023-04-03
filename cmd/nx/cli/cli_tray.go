package cli

import (
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/installer"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
)

func trayInstall() error {
	ret, err := installer.InstallTray()
	printOut(ret)
	return err
}

func trayUninstall() error {
	ret, err := shell.Launchctl.UninstallTrayAgent()
	printOut(ret)
	return err
}

func trayEnable() error {
	ret, err := shell.Launchctl.EnableTrayAgent()
	printOut(ret)
	return err
}

func trayDisable() error {
	ret, err := shell.Launchctl.DisableTrayAgent()
	printOut(ret)
	return err
}
