package cli

import (
	"go.digitalcircle.com.br/dc/netmux/cmd/nx/installer"
	"go.digitalcircle.com.br/dc/netmux/lib/cmd"
	"os"
)

func trayInstall() error {
	ret, err := installer.InstallTray()
	os.Stdout.WriteString(ret)
	return err
}
func trayUninstall() error {
	ret, err := cmd.LaunchCtlUninstallTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}
func trayEnable() error {
	ret, err := cmd.LaunchCtlEnableTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}
func trayDisable() error {
	ret, err := cmd.LaunchCtlDisableTrayAgent()
	os.Stdout.WriteString(ret)
	return err
}
