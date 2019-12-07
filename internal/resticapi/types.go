package resticapi

import "github.com/virvum/scmc/pkg/mycloud"

var validTypes = []string{"data", "index", "keys", "locks", "snapshots", "config"}

const (
	mimeTypeAPIV1 = "application/vnd.x.restic.rest.v1"
	mimeTypeAPIV2 = "application/vnd.x.restic.rest.v2"
)

type LogLevel uint

const (
	Debug LogLevel = 1
	Trace LogLevel = 2
)

type API struct {
	mc       map[string]*mycloud.MyCloud
	logLevel LogLevel
}
