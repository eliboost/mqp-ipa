// Package ldap configures and creates a UserGetter which is used to get the email based on the login name
package ldap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	directoryPath = "rest/v1/directory"

	roleBase   = "cn=%s,ou=roles,ou=%s,ou=solutions,ou=postfinance,o=post,c=ch"                       // role, application
	roleFilter = "scope=sub&filter=(member=cn=%s,ou=people,ou=postfinance,o=post,c=ch)&attributes=cn" // username
)

// basicAuth is used to authenticate with the LDAP REST service.
type basicAuth struct {
	username string
	password string
}

// Client defines options for service
type Client struct {
	auth       basicAuth
	baseURL    string
	httpClient *http.Client
}

// New inits service
func New(url string, auth basicAuth) RoleChecker {
	return &Client{
		auth:       auth,
		baseURL:    url,
		httpClient: &http.Client{},
	}
}

// HasAuthorRole checks if author has role
func (c Client) HasAuthorRole(application, role, username string) (bool, error) {
	filterQueryParam := fmt.Sprintf(roleFilter, username)
	rolePath := fmt.Sprintf(roleBase, role, application)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s?%s", c.baseURL, directoryPath, rolePath, filterQueryParam), http.NoBody)
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(c.auth.username, c.auth.password)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) //nolint:gosec // G704: false positive; URL is built from a configured base URL, not untrusted user input
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusNotAcceptable {
		// the search base entry does not exist
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("http request failed: %s", resp.Status)
	}

	result := searchQueryResult{}

	if err := getJSON(resp, &result); err != nil {
		return false, err
	}

	if result.TotalResults == 0 {
		return false, nil
	}

	if result.TotalResults != 1 {
		return false, errors.New("unexpected number of results. we expect only one user in result list")
	}

	return true, nil
}

func getJSON(resp *http.Response, target any) error {
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

type searchQueryResult struct {
	MsgType             string `json:"msgType"`
	TotalResults        int    `json:"totalResults"`
	SearchResultEntries []struct {
		Dn         string `json:"dn"`
		Attributes struct {
			Cn string `json:"cn"`
		} `json:"attributes"`
	} `json:"searchResultEntries"`
}

// RoleChecker defines functions
type RoleChecker interface {
	HasAuthorRole(application, role, username string) (bool, error)
}
