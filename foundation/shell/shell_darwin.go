package shell

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
)

// Provides API access.
var (
	Ifconfig  ifconfig
	Who       who
	Kill      kill
	Lsof      lsof
	Launchctl launchctl
)

// =============================================================================

// Ping performs network access to the specified address.
func Ping(addr string) (string, error) {
	return executeStr(fmt.Sprintf("ping %s", addr))
}

// Nmap preforms a network scan of the specified address.
func Nmap(addr string) (string, error) {
	return executeStr(fmt.Sprintf("nmap %s", addr))
}

// Netcat executes the netcat utility against the specified address.
func Netcat(h string) (string, error) {
	return executeStr(fmt.Sprintf("nc %s", h))
}

// =============================================================================

type ifconfig struct{}

// AddAlias adds the iface alias to the specified address.
func (ifconfig) AddAlias(iface string, ipaddr string) error {
	return execute(fmt.Sprintf("ifconfig %s alias %s", iface, ipaddr))
}

// RemoveAlias removes the iface alias to the specified address.
func (ifconfig) RemoveAlias(iface string, ipaddr string) error {
	return execute(fmt.Sprintf("ifconfig %s -alias %s", iface, ipaddr))
}

// =============================================================================

type who struct{}

// ConsoleUser returns the account who is currently logged in.
func (who) ConsoleUser() (string, error) {
	ret, err := executeStr("who | grep console")
	if err != nil {
		return "", fmt.Errorf("executeStr: %w", err)
	}

	parts := strings.Split(ret, " ")
	return parts[0], nil
}

// =============================================================================

type kill struct{}

// Kubectl kills any instance of kubectl if it is running.
func (kill) Kubectl() error {
	return execute("killall -9 kubectl")
}

// KubectlStr kills any instance of kubectl if it is running with output.
func KubectlStr() (string, error) {
	return executeStr("killall -9 kubectl")
}

// Pid kills the running instance of the specified pid.
func (kill) Pid(pid int) error {
	return execute(fmt.Sprintf("kill -9 %v", pid))
}

// =============================================================================

type lsof struct{}

// TcpConnsByPid lists all open connections for the specified pid.
func (lsof) TcpConnsByPid(pid int) (string, error) {
	return executeStr(fmt.Sprintf("lsof -p %v", pid))
}

// =============================================================================

type launchctl struct{}

// InstallDaemon installs the nx program.
func (launchctl) InstallDaemon() (string, error) {
	return executeStr("launchctl load /Library/LaunchDaemons/nx.plist")
}

// StartDaemon starts the nx program.
func (launchctl) StartDaemon() (string, error) {
	return executeStr("launchctl start nx")
}

// StopDaemon stops the nx program.
func (launchctl) StopDaemon() (string, error) {
	return executeStr("launchctl stop nx")
}

// UnistallDaemon uninstalls the nx program.
func (launchctl) UnistallDaemon() (string, error) {
	return executeStr("launchctl unload /Library/LaunchDaemons/nx.plist")
}

// InstallTrayAgent installs nx tray program.
func (launchctl) InstallTrayAgent() (string, error) {
	usr, err := getUnderlingUser()
	if err != nil {
		return "", fmt.Errorf("getUnderlingUser: %w", err)
	}

	return executeStr(fmt.Sprintf("launchctl load user/%s %s/Library/LaunchAgents/nx.tray.plist", usr.Uid, usr.HomeDir))
}

// StartTrayAgent starts the nx tray program.
func (launchctl) StartTrayAgent() (string, error) {
	return executeStr("launchctl start nx.tray")
}

// EnableTrayAgent enables the nx tray program.
func (launchctl) EnableTrayAgent() (string, error) {
	usr, err := getUnderlingUser()
	if err != nil {
		return "", fmt.Errorf("getUnderlingUser: %w", err)
	}

	return executeStr(fmt.Sprintf("launchctl enable user/%s/nx.tray", usr.Uid))
}

// DisableTrayAgent disables the nx tray program.
func (launchctl) DisableTrayAgent() (string, error) {
	usr, err := getUnderlingUser()
	if err != nil {
		return "", fmt.Errorf("getUnderlingUser: %w", err)
	}

	return executeStr(fmt.Sprintf("launchctl disable user/%s/nx.tray", usr.Uid))
}

// UninstallTrayAgent uninstalls the nx tray program.
func (launchctl) UninstallTrayAgent() (string, error) {
	usr, err := getUnderlingUser()
	if err != nil {
		return "", fmt.Errorf("getUnderlingUser: %w", err)
	}

	return executeStr(fmt.Sprintf("launchctl unload user/%s %s/Library/LaunchAgents/nx.tray.plist", usr.Uid, usr.HomeDir))
}

// =============================================================================

func getUnderlingUser() (*user.User, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	if usr.Username == "root" {
		if under := os.Getenv("SUDO_USER"); under != "" {
			usr, err := user.Lookup(under)
			if err != nil {
				return nil, err
			}

			return usr, nil
		}

		return nil, errors.New("unable to find root user")
	}

	return usr, nil
}
