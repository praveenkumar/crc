package preflight

import (
	"fmt"
	"strings"
)

var (
	daemonServiceScript = []string{
		`$ErrorActionPreference = "Stop"`,
		`$username = "%s"`,
		`$password = "%s"`,
		`$tempDir = "%s"`,
		`$crcExecutablePath = "%s"`,
		`$serviceName = "%s"`,
		`function CreateDaemonService()`,
		`{`,
		`   $secpass = (new-object System.Security.SecureString)`,
		//		`	$secPass = ConvertTo-SecureString $password -AsPlainText -Force`,
		`   $user = "DESKTOP-QAVB8EM\crc"`,
		`	$creds = New-Object pscredential ($user, $secPass)`,
		`	$params = @{`,
		`		Name = "$serviceName"`,
		`		BinaryPathName = "$crcExecutablePath"`,
		`		DisplayName = "$serviceName"`,
		`		StartupType = "Automatic"`,
		`		Description = "CodeReady Containers Daemon service for System Tray."`,
		`		Credential = $creds`,
		`	}`,
		`	New-Service @params`,
		`}`,
		`function StartDaemonService()`,
		`{`,
		`	Start-Service "$serviceName"`,
		`}`,
		`sc.exe stop "$serviceName"`,
		`sc.exe delete "$serviceName"`,
		`CreateDaemonService`,
		`StartDaemonService`,
		`New-Item -ItemType File -Path "$tempDir" -Name "success"`,
		`Set-Content -Path $tempDir\success "blah blah"`,
	}

	daemonServiceRemovalScript = []string{
		`$serviceName = "%s"`,
		`function DeleteDaemonService()`,
		`{`,
		`	sc.exe stop "$serviceName"`,
		`	sc.exe delete "$serviceName"`,
		`}`,
		`DeleteDaemonService`,
	}
)

func getDaemonServiceInstallationScriptTemplate() string {
	return strings.Join(daemonServiceScript, "\n")
}

func getDaemonServiceRemovalScriptTemplate() string {
	return strings.Join(daemonServiceRemovalScript, "\n")
}

func genDaemonServiceInstallScript(username, password, tempDirPath, daemonCmd, daemonServiceName string) string {
	return fmt.Sprintf(getDaemonServiceInstallationScriptTemplate(),
		username,
		password,
		tempDirPath,
		daemonCmd,
		daemonServiceName,
	)
}

func genDaemonServiceRemovalScript(daemonServiceName string) string {
	return fmt.Sprintf(getDaemonServiceRemovalScriptTemplate(),
		daemonServiceName,
	)
}
