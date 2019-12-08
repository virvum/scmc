package resticapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/virvum/scmc/pkg/logger"
	"github.com/virvum/scmc/pkg/mycloud"
)

var log logger.Log

// New creates a restic REST API resource.
func New(l logger.Log) *API {
	log = l

	return &API{
		mc: make(map[string]*mycloud.MyCloud),
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	if log.Level <= logger.Debug {
		requestDump, err := httputil.DumpRequest(r, log.Level <= logger.Trace)
		log.Debug("%s %s\n", r.Method, r.URL)

		if err != nil {
			log.Debug("httputil.DumpRequest: %v", err)
		} else {
			log.Debug("incoming request to %s:", r.URL)
			for _, line := range strings.Split(strings.TrimSpace(string(requestDump)), "\n") {
				log.Debug("> %s", line)
			}
		}
	}

	username, password, ok := r.BasicAuth()

	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if _, ok := a.mc[username]; !ok {
		// TODO we need to block other requests here

		mc, err := mycloud.New(username, password, log)
		if err != nil {
			log.Error("authorization failed: %s", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		a.mc[username] = mc
	}

	switch r.Header.Get("Accept") {
	case mimeTypeAPIV2:
		w.Header().Set("Content-Type", mimeTypeAPIV2)
	default:
		w.Header().Set("Content-Type", mimeTypeAPIV1)
	}

	fields := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// TODO this is dirty as fuck
	// Convert /foo/data/d14ae05f4385855567b1260997d84a8f8eae23cbe2c1d29b7a5e1b6313283004 to /foo/data/d1/d14ae05f4385855567b1260997d84a8f8eae23cbe2c1d29b7a5e1b6313283004
	if len(fields) == 3 && fields[len(fields)-2] == "data" {
		r.URL.Path = fmt.Sprintf("/%s/data/%s/%s",
			strings.Join(fields[0:len(fields)-2], "/"),
			fields[len(fields)-1][0:2],
			fields[len(fields)-1])
	}

	if strings.HasSuffix(r.URL.Path, "/") {
		switch r.Method {
		case http.MethodPost:
			if r.URL.Query().Get("create") != "true" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				err = fmt.Errorf("bad request to restic REST API")
			}

			err = a.create(username, w, r)
		case http.MethodDelete:
			err = a.delete(username, w, r)
		case http.MethodGet:
			err = a.list(username, w, r)
		default:
			w.WriteHeader(http.StatusNotImplemented)
			err = fmt.Errorf("not implemented in restic REST API")
		}
	} else {
		switch r.Method {
		case http.MethodHead:
			err = a.check(username, w, r)
		case http.MethodGet:
			err = a.get(username, w, r)
		case http.MethodPost:
			err = a.save(username, w, r)
		case http.MethodDelete:
			err = a.delete(username, w, r)
		default:
			w.WriteHeader(http.StatusNotImplemented)
			err = fmt.Errorf("not implemented in restic REST API")
		}
	}

	var (
		result    string
		httpRange string
	)

	switch err {
	case nil:
		result = fmt.Sprintf("\033[1;32m%s\033[0m", "OK")
	default:
		result = fmt.Sprintf("\033[1;31merror: %s\033[0m", err)
	}

	switch httpRange = r.Header.Get("Range"); {
	case httpRange == "":
	default:
		httpRange = " " + httpRange
	}

	log.Info("\033[1;34m%s %s\033[0m%s -> %s\n", r.Method, r.URL, httpRange, result)
}

// Creates the restic repository layout.
func (a *API) create(username string, w http.ResponseWriter, r *http.Request) error {
	if err := a.mc[username].CreateDirectory(r.URL.Path); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("internal server error: %v", err)
	}

	for _, d := range validTypes {
		if d == "config" {
			continue
		}

		if err := a.mc[username].CreateDirectory(fmt.Sprintf("%s%s/", r.URL.Path, d)); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return fmt.Errorf("internal server error: %v", err)
		}
	}

	for i := 0; i < 256; i++ {
		if err := a.mc[username].CreateDirectory(fmt.Sprintf("%sdata/%02x/", r.URL.Path, i)); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return fmt.Errorf("internal server error: %v", err)
		}
	}

	return nil
}

// Delete a directory and all of its contents or a file.
func (a *API) delete(username string, w http.ResponseWriter, r *http.Request) error {
	if err := a.mc[username].Delete([]string{r.URL.Path}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("internal server error: %v", err)
	}

	return nil
}

// Check whether a file exists and return its size in bytes in the Content-Length header.
func (a *API) check(username string, w http.ResponseWriter, r *http.Request) error {
	metadata, err := a.mc[username].Metadata(r.URL.Path)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return fmt.Errorf("not found: %v", err)
	}

	w.Header().Add("Content-Length", fmt.Sprint(metadata.Length))

	return nil
}

// Returns the content of the given file path.
func (a *API) get(username string, w http.ResponseWriter, r *http.Request) error {
	// TODO w.Header().Add("Content-Type", "binary/octet-stream")

	httpRange := r.Header.Get("Range")

	if err := a.mc[username].GetFile(r.URL.Path, w, httpRange); err != nil {
		return fmt.Errorf("a.mc.GetFile: %v", err)
	}

	return nil
}

// Saves the content of the request body as a file at the given path.
func (a *API) save(username string, w http.ResponseWriter, r *http.Request) error {
	if err := a.mc[username].CreateFile(r.URL.Path, r.Body); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("internal server error: %v", err)
	}

	return nil
}

// Returns a JSON array containing the names of all files stored at the given path.
func (a *API) list(username string, w http.ResponseWriter, r *http.Request) error {
	metadata, err := a.mc[username].Metadata(r.URL.Path)
	if err != nil {
		return fmt.Errorf("not found: %v", err)
	}

	var response []interface{}

	// TODO dirty dirty dirty
	// TODO If /foo/data/ is requested, list all blobs

	if strings.HasSuffix(strings.Trim(r.URL.Path, "/"), "/data") {
		for _, d := range metadata.Directories {
			m, err := a.mc[username].Metadata(strings.TrimRight(r.URL.Path, "/") + "/" + d.Name + "/")
			if err != nil {
				return fmt.Errorf("not found: %v", err)
			}

			switch r.Header.Get("Accept") {
			case mimeTypeAPIV2:
				for _, f := range m.Files {
					response = append(response, struct {
						Name string `json:"name"`
						Size uint64 `json:"size"`
					}{
						f.Name,
						f.Length,
					})
				}
			default:
				for _, f := range m.Files {
					response = append(response, f.Name)
				}
			}
		}
	} else {
		switch r.Header.Get("Accept") {
		case mimeTypeAPIV2:
			for _, f := range metadata.Files {
				response = append(response, struct {
					Name string `json:"name"`
					Size uint64 `json:"size"`
				}{
					f.Name,
					f.Length,
				})
			}
		default:
			for _, f := range metadata.Files {
				response = append(response, f.Name)
			}
		}
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("json.Marshal: %v", err)
	}

	w.Write(responseJSON)

	return nil
}
