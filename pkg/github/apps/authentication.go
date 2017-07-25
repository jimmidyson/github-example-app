// Copyright © 2017 Syndesis Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apps

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

const appsAcceptHeader = "application/vnd.github.machine-man-preview+json"

var apiBaseURL = strings.TrimSuffix(github.NewClient(http.DefaultClient).BaseURL.String(), "/")

// Transport provides a http.RoundTripper by wrapping an existing
// http.RoundTripper and provides GitHub App authentication as an
// installation.
//
// See https://developer.github.com/early-access/integrations/authentication/#as-an-installation
type Transport struct {
	BaseURL        string            // baseURL is the scheme and host for GitHub API, defaults to https://api.github.com
	tr             http.RoundTripper // tr is the underlying roundtripper being wrapped
	key            *rsa.PrivateKey   // key is the GitHub App's private key
	appID          int               // appID is the GitHub App's Installation ID
	installationID int               // installationID is the GitHub App's Installation ID
	authConfigurer func(*http.Request, string)

	mu    *sync.Mutex  // mu protects token
	token *accessToken // token is the installation's access token
}

// accessToken is an installation access token response from GitHub
type accessToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

var _ http.RoundTripper = &Transport{}

// NewAPITransportFromKeyFile returns a Transport for API use using a private key from file.
func NewAPITransportFromKeyFile(tr http.RoundTripper, appID, installationID int, privateKeyFile string) (*Transport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read private key")
	}
	return NewAPITransport(tr, appID, installationID, privateKey)
}

// NewAPITransport returns an Transport for API using private key. The key is parsed
// and if any errors occur the transport is nil and error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport is safe to be used concurrently.
func NewAPITransport(tr http.RoundTripper, appID, installationID int, privateKey []byte) (*Transport, error) {
	t := &Transport{
		tr:             tr,
		appID:          appID,
		installationID: installationID,
		BaseURL:        apiBaseURL,
		mu:             &sync.Mutex{},
		authConfigurer: func(req *http.Request, token string) {
			fmt.Printf("token %s\n", token)
			req.Header.Set("Authorization", "token "+token)
		},
	}
	var err error
	t.key, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse private key")
	}
	return t, nil
}

// NewGitTransportFromKeyFile returns a Transport for git use using a private key from file.
func NewGitTransportFromKeyFile(tr http.RoundTripper, appID, installationID int, privateKeyFile string) (*Transport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read private key")
	}
	return NewGitTransport(tr, appID, installationID, privateKey)
}

// NewGitTransport returns an Transport for git using private key. The key is parsed
// and if any errors occur the transport is nil and error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport is safe to be used concurrently.
func NewGitTransport(tr http.RoundTripper, appID, installationID int, privateKey []byte) (*Transport, error) {
	t := &Transport{
		tr:             tr,
		appID:          appID,
		installationID: installationID,
		BaseURL:        apiBaseURL,
		mu:             &sync.Mutex{},
		authConfigurer: func(req *http.Request, token string) {
			fmt.Printf("Using token %s\n", token)
			req.SetBasicAuth("x-access-token", token)
		},
	}
	var err error
	t.key, err = jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse private key")
	}
	return t, nil
}

// RoundTrip implements http.RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if t.token == nil || t.token.ExpiresAt.Add(-time.Minute).Before(time.Now()) {
		// Token is not set or expired/nearly expired, so refresh
		if err := t.refreshToken(); err != nil {
			t.mu.Unlock()
			return nil, errors.Wrapf(err, "could not refresh installation id %d's token", t.installationID)
		}
	}
	t.mu.Unlock()

	t.authConfigurer(req, t.token.Token)
	resp, err := t.tr.RoundTrip(req)
	return resp, err
}

func (t *Transport) refreshToken() error {
	// TODO these claims could probably be reused between installations before expiry
	claims := &jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		Issuer:    strconv.Itoa(t.appID),
	}
	bearer := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	ss, err := bearer.SignedString(t.key)
	if err != nil {
		return errors.Wrap(err, "could not sign jwt")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/installations/%d/access_tokens", t.BaseURL, t.installationID), nil)
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ss))
	req.Header.Set("Accept", appsAcceptHeader)

	client := &http.Client{Transport: t.tr}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "could not get access_tokens from GitHub API for installation ID %d", t.installationID)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.Errorf("received non 2xx response status %d (%q) when fetching %v. Response body: %s", resp.StatusCode, resp.Status, req.URL, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&t.token); err != nil {
		return err
	}

	return nil
}
