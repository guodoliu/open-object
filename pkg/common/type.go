package common

import "path/filepath"

const (
	ParamDriverName   = "driverName"
	DefaultEndpoint   = "unix://tmp/csi.sock"
	DefaultDriverName = "object.csi.gordon.com"

	HostDir             = "/host"
	ConfigDir           = "/etc/open-object"
	ConnectorSocketName = "connector.sock"

	NsenterCmd = "/bin/nsenter --mount=/proc/1/ns/mnt -ipc=/proc/1/ns/ipc --net=/proc/1/ns/net --uts=/proc/1/ns/uts"

	// VolumeOperationAlreadyExists is message fmt returned to CO when there is another in-flight call on the given volumeID
	VolumeOperationAlreadyExists = "An operation with the given volume=%q is already in progress"
)

var (
	// ConnectorWorkPath workspace
	ConnectorWorkPath = "./"
	// ConnectorSocketPath socket path
	ConnectorSocketPath = filepath.Join(ConfigDir, "connector.sock")
	// ConnectorLogFilename name of log file
	ConnectorLogFilename = filepath.Join(ConfigDir, "connector.log")
	// ConnectorPIDFilename name of pid file
	ConnectorPIDFilename = filepath.Join(ConfigDir, "connector.pid")
)
