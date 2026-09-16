// Package nameservers provides functionality for working with glue records:
// the nameserver hosts a registry publishes addresses for, so that a zone can
// be served by nameservers named inside itself.
//
// This is a different object from a nameserver *group* (see the nsgroups
// package). A group names which hosts answer for a domain; a glue record gives
// the registry the addresses of one host, which is what makes an in-bailiwick
// name resolvable at all.
package nameservers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Nameserver is one glue record: a host name and the addresses published for it.
// At least one of IP and IP6 must be set.
type Nameserver struct {
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
	IP6  string `json:"ip6,omitempty"`
}

// response is the envelope Openprovider wraps every reply in. `code` is zero on
// success and carries the error otherwise, whatever the HTTP status says.
type response struct {
	Code int             `json:"code"`
	Desc string          `json:"desc"`
	Data json.RawMessage `json:"data"`
}

// ErrNotFound reports that the named glue record does not exist. Callers use it
// to tell a deleted record apart from a failed read.
var ErrNotFound = fmt.Errorf("nameserver not found")

// do sends the request and unwraps the envelope. It returns the `data` member
// so the caller can decode whichever shape the endpoint answers with.
func do(c *client.Client, req *http.Request) (json.RawMessage, error) {
	resp, err := c.Do(req)
	if err != nil {
		// `client.Client.Do` turns any non-2xx status into an error and closes
		// the body before returning, so a 404 is told apart here, from the
		// status alone, rather than from a body that is no longer there to read.
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope response
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not read the reply (status %d): %w", resp.StatusCode, err)
	}

	// A non-zero `code` is an API error even on a 2xx status, so it is
	// checked on its own. Openprovider reports a missing object as code 399.
	// `resp.StatusCode` is 2xx whenever `err` is nil, so no status check is
	// needed below this point.
	if envelope.Code != 0 {
		if envelope.Code == 399 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("openprovider error %d: %s", envelope.Code, envelope.Desc)
	}

	return envelope.Data, nil
}

// Get retrieves one glue record by host name.
//
// Endpoint: GET /v1beta/dns/nameservers/{name}
func Get(c *client.Client, name string) (*Nameserver, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/dns/nameservers/%s", c.BaseURL, name), nil)
	if err != nil {
		return nil, err
	}

	data, err := do(c, req)
	if err != nil {
		return nil, err
	}

	var ns Nameserver
	if err := json.Unmarshal(data, &ns); err != nil {
		return nil, err
	}

	// The endpoint answers 200 with an empty object for a name it does not
	// hold, so an absent name is the second way a record reads as missing.
	if ns.Name == "" {
		return nil, ErrNotFound
	}

	return &ns, nil
}
