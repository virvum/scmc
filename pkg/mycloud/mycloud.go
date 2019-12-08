// Package mycloud implements a library which can be used to interact with Swisscom's myCloud service.
package mycloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"path/filepath"
	"strings"

	"github.com/virvum/scmc/pkg/logger"
)

const (
	userAgent      = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.87 Safari/537.36"
	identityServer = "https://identity.prod.mdl.swisscom.ch"
	storageServer  = "https://storage.prod.mdl.swisscom.ch"
	pathPrefix     = "/Drive"
)

var log logger.Log

// TODO automatically re-authenticate, when token isn't valid anymore

// New creates a new myCloud instance. This function will automatically authenticate the given user.
func New(username string, password string, l logger.Log) (*MyCloud, error) {
	log = l

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	mc := &MyCloud{
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				log.Debug("redirect: %v\n", req.URL)
				return nil
			},
		},
	}

	if err := mc.authenticate(username, password); err != nil {
		return mc, fmt.Errorf("mc.authenticate: %v", err)
	}

	return mc, nil
}

// Request is used to access a myCloud resource in a generic way.
// Important: response.Body.Close() required, when r.Result is not set.
func (mc *MyCloud) Request(r Request) error {
	client := &http.Client{}

	request, err := http.NewRequest(r.Method, r.Server+"/"+r.Action, r.Reader)
	if err != nil {
		return fmt.Errorf("http.NewRequest: %v", err)
	}

	if r.Path != "" {
		log.Debug("setting query string for path '%s'", r.Path)
		q := request.URL.Query()
		q.Add("p", base64.StdEncoding.EncodeToString([]byte(pathPrefix+r.Path)))
		request.URL.RawQuery = q.Encode()
	}

	if mc.accessToken == "" {
		return fmt.Errorf("no access token (bearer) found")
	}

	request.Header.Add("Authorization", "Bearer "+mc.accessToken)
	request.Header.Add("User-Agent", userAgent)
	request.Header.Add("Origin", "https://www.mycloud.ch/")
	request.Header.Add("Referer", "https://www.mycloud.ch/")

	if r.HTTPRange != "" {
		log.Debug("setting HTTP range: %s", r.HTTPRange)
		request.Header.Add("Range", r.HTTPRange)
	}

	// TODO
	// request.Header.Set("Accept-Encoding", "identity")
	// request.TransferEncoding = []string{"identity"}

	if r.ContentType != "" {
		request.Header.Add("Content-Type", r.ContentType)
	} else if r.Reader != nil {
		request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	}

	if log.IsDebug() {
		requestDump, err := httputil.DumpRequestOut(request, log.IsTrace())
		if err != nil {
			log.Debug("httputil.DumpRequest: %v", err)
		} else {
			log.Debug("outgoing request to %s [%s]:", request.URL, r.Path)
			for _, line := range strings.Split(strings.TrimSpace(string(requestDump)), "\n") {
				log.Debug("> %s", line)
			}
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("client.Do: %v", err)
	}

	if log.IsDebug() {
		requestDump, err := httputil.DumpResponse(response, log.IsTrace())
		if err != nil {
			log.Debug("httputil.DumpRequest: %v", err)
		} else {
			log.Debug("response from %s [%s]:", request.URL, r.Path)
			for _, line := range strings.Split(strings.TrimSpace(string(requestDump)), "\n") {
				log.Debug("> %s", line)
			}
		}
	}

	if r.Response != nil {
		*r.Response = response
	} else if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("ioutil.ReadAll: %v", err)
		} else {
			for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
				log.Error("response body: %s", line)
			}
		}

		return fmt.Errorf("got status code %d instead of 200", response.StatusCode)
	}

	if r.Result != nil {
		defer response.Body.Close()

		decoder := json.NewDecoder(response.Body)

		for {
			if err := decoder.Decode(r.Result); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("json.Decode: %v", err)
			}
		}
	}

	return nil
}

// AccessToken returns access token used to access myCloud.
func (mc *MyCloud) AccessToken() string {
	return mc.accessToken
}

// Identity returns user account identity information.
func (mc *MyCloud) Identity() (*IdentityResponse, error) {
	var r IdentityResponse

	if err := mc.Request(Request{
		Method: "GET",
		Server: identityServer,
		Action: "me",
		Result: &r,
	}); err != nil {
		return nil, fmt.Errorf("mc.Request: %v", err)
	}

	return &r, nil
}

// Usage returns account usage information.
func (mc *MyCloud) Usage() (*UsageResponse, error) {
	var r UsageResponse

	if err := mc.Request(Request{
		Method: "GET",
		Server: storageServer,
		Action: "usage",
		Result: &r,
	}); err != nil {
		return nil, fmt.Errorf("mc.Request: %v", err)
	}

	return &r, nil
}

// Metadata fetches metadata for the given file or directory. Directories must end with a slash.
func (mc *MyCloud) Metadata(path string) (*MetadataResponse, error) {
	var r MetadataResponse

	if err := mc.Request(Request{
		Method: "GET",
		Server: storageServer,
		Action: "metadata",
		Path:   path,
		Result: &r,
	}); err != nil {
		return nil, fmt.Errorf("mc.Request: %v", err)
	}

	return &r, nil
}

// CreateDirectory creates a directory with all parent directories. Specified directory path must end with a slash.
func (mc *MyCloud) CreateDirectory(path string) error {
	var r CreateDirectoryResponse

	if !strings.HasSuffix(path, "/") {
		return fmt.Errorf("path must end with a slash: %v", path)
	}

	if err := mc.Request(Request{
		Method: "PUT",
		Server: storageServer,
		Action: "object",
		Path:   path,
		Result: &r,
	}); err != nil {
		return fmt.Errorf("mc.Request: %v", err)
	}

	if r.Name != filepath.Base(path) {
		return fmt.Errorf("invalid name returned: %v", r.Name)
	}

	return nil
}

// Delete deletes files or directories. Directories will be deleted recursively. Specified directory paths must end with a slash.
// TODO: add option to actually delete files, not moving them to the trash
func (mc *MyCloud) Delete(paths []string) error {
	var (
		requestBody DeleteRequest
		r           DeleteResponse
	)

	for _, p := range paths {
		requestBody.Items = append(requestBody.Items, pathPrefix+p)
	}

	reqJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("json.Marshal: %v", err)
	}

	if err := mc.Request(Request{
		Method: "PUT",
		Server: storageServer,
		Action: "trash/items",
		Reader: bytes.NewBuffer(reqJSON),
		Result: &r,
	}); err != nil {
		return fmt.Errorf("mc.Request: %v", err)
	}

	// We trust myCloud (sigh...)
	// TODO do a more sophisticated check
	if len(r.Failed) > 0 {
		return fmt.Errorf("deletion not completed for the following files: %v", r.Failed)
	}

	return nil
}

// CreateFile uploads a file.
func (mc *MyCloud) CreateFile(path string, dataReader io.Reader) error {
	var r MetadataResponse

	if err := mc.Request(Request{
		Method:      "PUT",
		Server:      storageServer,
		Action:      "object",
		Path:        path,
		Reader:      dataReader,
		ContentType: "application/octet-stream",
		Result:      &r,
	}); err != nil {
		return fmt.Errorf("mc.Request: %v", err)
	}

	if r.Name != filepath.Base(path) {
		return fmt.Errorf("invalid metadata returned")
	}

	return nil
}

// GetFile downloads a file.
// TODO replace httpRange with a httpRange range type
func (mc *MyCloud) GetFile(path string, dataWriter io.Writer, httpRange string) error {
	var response *http.Response

	if err := mc.Request(Request{
		Method:    "GET",
		Server:    storageServer,
		Action:    "object",
		Path:      path,
		HTTPRange: httpRange,
		Response:  &response,
	}); err != nil {
		return fmt.Errorf("mc.Request: %v", err)
	}

	defer response.Body.Close()

	if httpRange != "" {
		if response.StatusCode != 206 {
			// TODO output response body
			return fmt.Errorf("got status code %d (expected 206)", response.StatusCode)
		}
	} else if response.StatusCode != 200 {
		// TODO output response body
		return fmt.Errorf("got status code %d (expected 200)", response.StatusCode)
	}

	if _, err := io.Copy(dataWriter, response.Body); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	return nil
}
