package preflight

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"syscall"

	crcErrors "github.com/code-ready/crc/pkg/crc/errors"
	"github.com/code-ready/crc/pkg/crc/logging"
	"github.com/code-ready/crc/pkg/crc/network"
	crcos "github.com/code-ready/crc/pkg/os"
	"github.com/code-ready/crc/pkg/os/linux"
)

func libvirtPreflightChecks(distro *linux.OsRelease) []Check {
	checks := []Check{
		{
			configKeySuffix:  "check-virt-enabled",
			checkDescription: "Checking if Virtualization is enabled",
			check:            checkVirtualizationEnabled,
			fixDescription:   "Setting up virtualization",
			fix:              fixVirtualizationEnabled,
		},
		{
			configKeySuffix:  "check-kvm-enabled",
			checkDescription: "Checking if KVM is enabled",
			check:            checkKvmEnabled,
			fixDescription:   "Setting up KVM",
			fix:              fixKvmEnabled,
		},
		{
			configKeySuffix:  "check-libvirt-installed",
			checkDescription: "Checking if libvirt is installed",
			check:            checkLibvirtInstalled,
			fixDescription:   "Installing libvirt service and dependencies",
			fix:              fixLibvirtInstalled(distro),
		},
		{
			configKeySuffix:  "check-user-in-libvirt-group",
			checkDescription: "Checking if user is part of libvirt group",
			check:            checkUserPartOfLibvirtGroup,
			fixDescription:   "Adding user to libvirt group",
			fix:              fixUserPartOfLibvirtGroup(distro),
		},
		{
			configKeySuffix:  "check-libvirt-running",
			checkDescription: "Checking if libvirt daemon is running",
			check:            checkLibvirtServiceRunning,
			fixDescription:   "Starting libvirt service",
			fix:              fixLibvirtServiceRunning,
		},
		{
			configKeySuffix:  "check-libvirt-version",
			checkDescription: "Checking if a supported libvirt version is installed",
			check:            checkLibvirtVersion,
			fixDescription:   "Installing a supported libvirt version",
			fix:              fixLibvirtVersion,
		},
		{
			configKeySuffix:  "check-libvirt-driver",
			checkDescription: "Checking if crc-driver-libvirt is installed",
			check:            checkMachineDriverLibvirtInstalled,
			fixDescription:   "Installing crc-driver-libvirt",
			fix:              fixMachineDriverLibvirtInstalled,
		},
		{
			cleanupDescription: "Removing the crc VM if exists",
			cleanup:            removeCrcVM,
			flags:              CleanUpOnly,
		},
	}
	if distroIsLike(distro, linux.Ubuntu) {
		checks = append(checks, ubuntuPreflightChecks...)
	}
	return checks
}

var libvirtNetworkPreflightChecks = [...]Check{
	{
		configKeySuffix:    "check-crc-network",
		checkDescription:   "Checking if libvirt 'crc' network is available",
		check:              checkLibvirtCrcNetworkAvailable,
		fixDescription:     "Setting up libvirt 'crc' network",
		fix:                fixLibvirtCrcNetworkAvailable,
		cleanupDescription: "Removing 'crc' network from libvirt",
		cleanup:            removeLibvirtCrcNetwork,
	},
	{
		configKeySuffix:  "check-crc-network-active",
		checkDescription: "Checking if libvirt 'crc' network is active",
		check:            checkLibvirtCrcNetworkActive,
		fixDescription:   "Starting libvirt 'crc' network",
		fix:              fixLibvirtCrcNetworkActive,
	},
}

var vsockPreflightChecks = Check{
	configKeySuffix:    "check-vsock",
	checkDescription:   "Checking if vsock is correctly configured",
	check:              checkVsock,
	fixDescription:     "Checking if vsock is correctly configured",
	fix:                fixVsock,
	cleanupDescription: "Removing vsock configuration",
	cleanup:            removeVsockCrcSettings,
}

const (
	vsockUdevRulesPath          = "/usr/lib/udev/rules.d/99-crc-vsock.rules"
	vsockModuleAutoLoadConfPath = "/etc/modules-load.d/vhost_vsock.conf"
)

func checkVsock() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	getcap, _, err := crcos.RunWithDefaultLocale("getcap", executable)
	if err != nil {
		return err
	}
	if !strings.Contains(string(getcap), "cap_net_bind_service+eip") {
		return fmt.Errorf("capabilities are not correct for %s", executable)
	}
	info, err := os.Stat("/dev/vsock")
	if err != nil {
		return err
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		group, err := user.LookupGroupId(fmt.Sprint(stat.Gid))
		if err != nil {
			return err
		}
		if group.Name != "libvirt" {
			return errors.New("/dev/vsock is not in the right group")
		}
	} else {
		return errors.New("cannot cast info")
	}
	if info.Mode()&0060 == 0 {
		return errors.New("/dev/vsock doesn't have the right permissions")
	}
	return nil
}

func fixVsock() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	_, _, err = crcos.RunWithPrivilege("setcap cap_net_bind_service=+eip", "setcap", "cap_net_bind_service=+eip", executable)
	if err != nil {
		return err
	}

	udevRule := `KERNEL=="vsock", MODE="0660", OWNER="root", GROUP="libvirt"`
	err = crcos.WriteToFileAsRoot("Create udev rule for /dev/vsock", udevRule, vsockUdevRulesPath, 0644)
	if err != nil {
		return err
	}
	err = crcos.WriteToFileAsRoot(fmt.Sprintf("Create file %s", vsockModuleAutoLoadConfPath), "vhost_vsock", vsockModuleAutoLoadConfPath, 0644)
	if err != nil {
		return err
	}
	_, _, err = crcos.RunWithPrivilege("modprobe vhost_vsock", "modprobe", "vhost_vsock")
	if err != nil {
		return err
	}
	return nil
}

func removeVsockCrcSettings() error {
	var mErr crcErrors.MultiError
	_, _, err := crcos.RunWithPrivilege(fmt.Sprintf("rm %s", vsockUdevRulesPath), "rm", "-f", vsockUdevRulesPath)
	if err != nil {
		mErr.Collect(err)
	}
	_, _, err = crcos.RunWithPrivilege(fmt.Sprintf("rm %s", vsockModuleAutoLoadConfPath), "rm", "-f", vsockModuleAutoLoadConfPath)
	if err != nil {
		mErr.Collect(err)
	}
	if len(mErr.Errors) == 0 {
		return nil
	}
	return mErr
}

func getAllPreflightChecks() []Check {
	checks := getPreflightChecksForDistro(distro(), network.DefaultMode)
	checks = append(checks, vsockPreflightChecks)
	return checks
}

func getPreflightChecks(_ bool, networkMode network.Mode) []Check {
	return getPreflightChecksForDistro(distro(), networkMode)
}

func getNetworkChecksForDistro(distro *linux.OsRelease, networkMode network.Mode) []Check {
	var checks []Check

	if networkMode == network.VSockMode {
		return append(checks, vsockPreflightChecks)
	}

	switch {
	default:
		logging.Warnf("distribution-specific preflight checks are not implemented for '%s'", distro.ID)
		fallthrough
	case distroIsLike(distro, linux.Ubuntu), distroIsLike(distro, linux.Fedora):
		checks = append(checks, nmPreflightChecks[:]...)
		if usesSystemdResolved(distro) {
			checks = append(checks, systemdResolvedPreflightChecks[:]...)
		} else {
			checks = append(checks, dnsmasqPreflightChecks[:]...)
		}
	}

	return checks
}

func getPreflightChecksForDistro(distro *linux.OsRelease, networkMode network.Mode) []Check {
	var checks []Check
	checks = append(checks, nonWinPreflightChecks[:]...)
	checks = append(checks, genericPreflightChecks[:]...)
	checks = append(checks, libvirtPreflightChecks(distro)...)
	networkChecks := getNetworkChecksForDistro(distro, networkMode)
	checks = append(checks, networkChecks...)
	if networkMode == network.DefaultMode {
		checks = append(checks, libvirtNetworkPreflightChecks[:]...)
	}
	return checks
}

func usesSystemdResolved(distro *linux.OsRelease) bool {
	switch {
	case distroIsLike(distro, linux.Ubuntu):
		return true
	case distro.ID == linux.Fedora:
		return distro.VersionID >= "33"
	default:
		return false
	}
}

func distroIsLike(osRelease *linux.OsRelease, osType linux.OsType) bool {
	if osRelease == nil {
		return false
	}
	if osRelease.ID == osType {
		return true
	}

	for _, id := range osRelease.GetIDLike() {
		if id == osType {
			return true
		}
	}

	return false
}

func distro() *linux.OsRelease {
	distro, err := linux.GetOsRelease()
	if err != nil {
		logging.Errorf("cannot get distribution name: %v", err)
		return &linux.OsRelease{
			ID: "unknown",
		}
	}
	return distro
}
