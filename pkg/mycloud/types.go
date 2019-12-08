package mycloud

import (
	"io"
	"net/http"
	"time"
)

// MyCloud represents a connection instance to myCloud.
type MyCloud struct {
	client      *http.Client
	authState   map[string]interface{}
	accessToken string
}

// Request represents request options.
type Request struct {
	Method      string
	Server      string
	Action      string
	Path        string
	Result      interface{}
	Response    **http.Response
	Reader      io.Reader
	ContentType string
	HTTPRange   string

	// TODO QueryString string // conflicts with `Path`
}

// IdentityResponse represents the data returned by the "identity" API call to myCloud.
type IdentityResponse struct {
	Identifier           string
	FirstName            string
	LastName             string
	UserName             string
	Email                string
	EmailConfirmed       bool
	PhoneNumber          string
	PhoneNumberConfirmed bool
	TermsAccepted        bool
	AnalyticsIdentifier  string
	Subscription         struct {
		Name                 string
		Identifier           string
		MaxFileSize          uint64
		IsUpgradable         bool
		IsDowngradable       bool
		Timestamp            string
		Created              string
		Reference            string
		ReferenceName        string
		ReferenceDescription string
	}
	Editions            interface{}
	LoginProviderBearer string
	HashID              string
}

// UsageResponse represents the data returned by the "usage" API call to myCloud.
type UsageResponse struct {
	BackupBytes    uint64
	DocumentsBytes uint64
	DriveBytes     uint64
	MoviesBytes    uint64
	MusicBytes     uint64
	PhotosBytes    uint64
	TVBytes        uint64
	TotalBytes     uint64
}

// CreateDirectoryResponse represents the data returned when creating a new directory on myCloud.
type CreateDirectoryResponse struct {
	Name             string
	Path             string
	CreationTime     string
	ModificationTime string
}

// MetadataResponse represents the data returned by the "metadata" API call to myCloud.
type MetadataResponse struct {
	Length           uint64 // only defined for file response
	Etag             string // only defined for file response
	Mime             string // only defined for file response
	Extension        string // only defined for file response
	Name             string
	Path             string
	CreationTime     time.Time
	ModificationTime time.Time
	Files            []struct {
		Name             string
		Path             string
		Etag             string
		Mime             string
		Length           uint64
		CreationTime     time.Time
		ModificationTime time.Time
		Extension        string
	}
	Directories []struct {
		Name             string
		Path             string
		CreationTime     time.Time
		ModificationTime time.Time
	}
}

// UploadResponse represents the data returned when uploading a file to myCloud.
type UploadResponse struct {
	CreationTime     string
	ModificationTime string
	Path             string
	Length           uint64
}

// DeleteRequest represents the items to be deleted via the myCloud API.
type DeleteRequest struct {
	Items []string
}

// DeleteResponse represents the data returned when deleting files or directories on myCloud.
type DeleteResponse struct {
	Completed []string
	Failed    []string
}
