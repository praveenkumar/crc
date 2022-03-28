package cmd

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
	"github.com/code-ready/crc/pkg/crc/constants"
	"github.com/code-ready/crc/pkg/crc/logging"
	crcversion "github.com/code-ready/crc/pkg/crc/version"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
)

func vsockListener() (net.Listener, error) {
	ln, err := transport.Listen(transport.DefaultURL)
	logging.Infof("listening %s", transport.DefaultURL)
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func httpListener() (net.Listener, error) {
	// see https://github.com/moby/moby/blob/46cdcd206c56172b95ba5c77b827a722dab426c5/daemon/listeners/listeners_windows.go
	// allow Administrators and SYSTEM, plus whatever additional users or groups were specified
	sddl := "D:P(A;;GA;;;BA)(A;;GA;;;SY)"
	sid, err := winio.LookupSidByName("crc-users")
	if err != nil {
		return nil, err
	}
	sddl += fmt.Sprintf("(A;;GRGW;;;%s)", sid)
	ln, err := winio.ListenPipe(constants.DaemonHTTPNamedPipe, &winio.PipeConfig{
		SecurityDescriptor: sddl,  // Administrators and system
		MessageMode:        true,  // Use message mode so that CloseWrite() is supported
		InputBufferSize:    65536, // Use 64kB buffers to improve performance
		OutputBufferSize:   65536,
	})
	logging.Infof("listening %s", constants.DaemonHTTPNamedPipe)
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func checkIfDaemonIsRunning() (bool, error) {
	return checkDaemonVersion()
}

func daemonNotRunningMessage() string {
	if crcversion.IsInstaller() {
		return "Is CodeReady Containers tray application running? Cannot reach daemon API"
	}
	return genericDaemonNotRunningMessage
}

func startupDone() {
}
