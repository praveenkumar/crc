package cmd

import (
	"net"
	"os"

	"github.com/code-ready/crc/pkg/crc/constants"
	"github.com/code-ready/crc/pkg/crc/logging"
	"github.com/code-ready/crc/pkg/os/launchd"
)

func vsockListener() (net.Listener, error) {
	_ = os.Remove(constants.TapSocketPath)
	ln, err := net.Listen("unix", constants.TapSocketPath)
	logging.Infof("listening %s", constants.TapSocketPath)
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func httpListener() (net.Listener, error) {
	_ = os.Remove(constants.DaemonHTTPSocketPath)
	ln, err := net.Listen("unix", constants.DaemonHTTPSocketPath)
	logging.Infof("listening %s", constants.DaemonHTTPSocketPath)
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func checkIfDaemonIsRunning() (bool, error) {
	if launchd.PlistExists(constants.DaemonAgentLabel) {
		/* detect if the daemon is being started by launchd,
		* and socket activation is in use. In this scenario,
		* trying to send an HTTP version check on the daemon
		* HTTP socket would hang as the socket is listening for
		* connections but is not setup to handle them yet.
		 */
		return false, nil
	}

	return checkDaemonVersion()
}

func daemonNotRunningMessage() string {
	return genericDaemonNotRunningMessage
}

func startupDone() {
}
