package mycloud

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/virvum/scmc/pkg/logger"

	"github.com/google/uuid"
)

func (mc *MyCloud) setAuthState(auth_state string) error {
	jsonData, err := base64.StdEncoding.DecodeString(auth_state)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonData, &mc.authState); err != nil {
		return err
	}

	return nil
}

func (mc *MyCloud) getAuthState() string {
	jsonData, err := json.Marshal(mc.authState)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(jsonData)
}

// TODO merge with mycloud.Request?
func (mc *MyCloud) srequest(method string, uri string, qs map[string]string, data map[string]string) (*http.Response, error) {
	var request *http.Request
	var err error

	if data == nil {
		request, err = http.NewRequest(method, uri, nil)
	} else {
		form := url.Values{}

		for k, v := range data {
			form.Add(k, v)
		}

		request, err = http.NewRequest(method, uri, strings.NewReader(form.Encode()))
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, err
	}

	if len(qs) > 0 {
		q := request.URL.Query()

		for k, v := range qs {
			q.Add(k, v)
		}

		request.URL.RawQuery = q.Encode()
	}

	if log.IsDebug() {
		requestDump, err := httputil.DumpRequestOut(request, log.Level <= logger.Trace)
		if err != nil {
			log.Debug("httputil.DumpRequest: %v", err)
		} else {
			log.Debug("outgoing request to %s:", request.URL)
			for _, line := range strings.Split(strings.TrimSpace(string(requestDump)), "\n") {
				log.Debug("> %s", line)
			}
		}
	}

	response, err := mc.client.Do(request)
	if err != nil {
		return nil, err
	}

	if log.IsDebug() {
		requestDump, err := httputil.DumpResponse(response, log.Level <= logger.Trace)
		if err != nil {
			log.Debug("httputil.DumpRequest: %v", err)
		} else {
			log.Debug("response from %s:", request.URL)
			for _, line := range strings.Split(strings.TrimSpace(string(requestDump)), "\n") {
				log.Debug("> %s", line)
			}
		}
	}

	return response, nil
}

// Authenticate with Swisscom myCloud.
//
// Please mote that this authentication procedure has been reverse-engineered,
// so it might not be that perfect after all.
func (mc *MyCloud) authenticate(username string, password string) error {
	var (
		err    error
		r      *http.Response
		params url.Values
		u      *url.URL
	)

	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("uuid.NewUUID: %v", err)
	}

	r, err = mc.srequest("GET", "https://support.prod.mdl.swisscom.ch/login", map[string]string{
		"client_id":        id.String(),
		"response_type":    "token",
		"redirect_uri":     "https://www.mycloud.ch/login",
		"application_type": "web",
		"state":            "IiI=", // base64('""')
	}, nil)
	if err != nil {
		return fmt.Errorf("mc.srequest: %v", err)
	}

	params, err = url.ParseQuery(r.Request.URL.RawQuery)
	if err != nil {
		return fmt.Errorf("url.ParseQuery: %v", err)
	}

	rurl := params.Get("RURL")
	if rurl == "" {
		return fmt.Errorf("params.Get: 'RURL' is empty")
	}

	u, err = url.Parse(rurl)
	if err != nil {
		return fmt.Errorf("url.Parse: %v", err)
	}

	params, err = url.ParseQuery(u.RawQuery)
	if err != nil {
		return fmt.Errorf("url.ParseQuery: %v", err)
	}

	authState := params.Get("auth_state")
	if authState == "" {
		return fmt.Errorf("params.Get: 'auth_state' is empty")
	}

	// Append "=" to the auth_state base64 string, otherwise we get
	// the error "illegal base64 data at input byte 92".
	authState += "="

	err = mc.setAuthState(authState)
	if err != nil {
		return fmt.Errorf("mc.setAuthState: %v", err)
	}

	mc.authState["providedUserId"] = username

	r, err = mc.srequest("GET", "https://identity-sc.prod.mdl.swisscom.ch/login", map[string]string{
		"type":       "login",
		"auth_state": mc.getAuthState(),
	}, nil)
	if err != nil {
		return fmt.Errorf("mc.srequest: %v", err)
	}

	params, err = url.ParseQuery(r.Request.URL.RawQuery)
	if err != nil {
		return fmt.Errorf("url.ParseQuery: %v", err)
	}

	r, err = mc.srequest("POST", "https://login.sso.bluewin.ch/login", map[string]string{
		"SNA":  "mycloud",
		"RURL": params.Get("RURL"),
		"UN":   username,
	}, map[string]string{
		"username": username,
		"p":        "true",
		"password": password,
		"anmelden": "", // this is actually required...
	})
	if err != nil {
		return fmt.Errorf("mc.srequest: %v", err)
	}

	params, err = url.ParseQuery(r.Request.URL.RawQuery)

	accessToken := params.Get("access_token")
	if accessToken == "" {
		return errors.New("no access token was returned")
	}

	// In the returned base64 access token the plus character has been replaced with a space character.
	mc.accessToken = strings.Replace(accessToken, " ", "+", -1)

	return nil
}
