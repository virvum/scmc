package resticapi

import "github.com/virvum/scmc/pkg/mycloud"

var validTypes = []string{"data", "index", "keys", "locks", "snapshots", "config"}

const (
	mimeTypeAPIV1 = "application/vnd.x.restic.rest.v1"
	mimeTypeAPIV2 = "application/vnd.x.restic.rest.v2"
)

// API represents an API object.
type API struct {
	mc map[string]*mycloud.MyCloud
}
