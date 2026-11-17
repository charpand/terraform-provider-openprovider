// Package domains provides functionality for working with domains.
package domains

import (
	"strings"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// Nameserver represents a domain nameserver.
type Nameserver struct {
	Name  string `json:"name"`
	IP    string `json:"ip,omitempty"`
	IP6   string `json:"ip6,omitempty"`
	SeqNr int    `json:"seq_nr,omitempty"`
}

// DnssecKey represents a DNSSEC key/DS record.
type DnssecKey struct {
	Alg      int    `json:"alg"`
	Flags    int    `json:"flags"`
	Protocol int    `json:"protocol"`
	PubKey   string `json:"pub_key"`
	// Readonly indicates if this key is read-only and managed by the registry (1) or can be modified (0).
	// This field is typically set by the API when retrieving existing keys.
	Readonly int `json:"readonly,omitempty"`
}

// Domain represents a domain entity.
type Domain struct {
	ID                    int          `json:"id"`
	ActiveDate            string       `json:"active_date"`
	AdminHandle           string       `json:"admin_handle"`
	AuthCode              string       `json:"auth_code"`
	Autorenew             string       `json:"autorenew"`
	BillingHandle         string       `json:"billing_handle"`
	CanRenew              bool         `json:"can_renew"`
	Comments              string       `json:"comments,omitempty"`
	IsPrivateWhoisEnabled bool         `json:"is_private_whois_enabled,omitempty"`
	IsPrivateWhoisAllowed bool         `json:"is_private_whois_allowed,omitempty"`
	CreationDate          string       `json:"creation_date"`
	ExpirationDate        string       `json:"expiration_date"`
	IsAbusive             bool         `json:"is_abusive"`
	IsLocked              bool         `json:"is_locked"`
	LastChanged           string       `json:"last_changed"`
	OrderDate             string       `json:"order_date"`
	OwnerHandle           string       `json:"owner_handle"`
	Status                string       `json:"status"`
	TechHandle            string       `json:"tech_handle"`
	Nameservers           []Nameserver `json:"name_servers,omitempty"`
	NSGroup               string       `json:"ns_group,omitempty"`
	DnssecKeys            []DnssecKey  `json:"dnssec_keys,omitempty"`
	IsDnssecEnabled       bool         `json:"is_dnssec_enabled,omitempty"`
	Domain                struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
	} `json:"domain"`
}

// ListDomainsResponse represents a response from the domains listing endpoint.
type ListDomainsResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Results []Domain `json:"results"`
		Total   int      `json:"total"`
	} `json:"data"`
}

// List retrieves every domain from the Openprovider API, paging through the
// account's full listing rather than handing back only the API's first page.
func List(c *client.Client) ([]Domain, error) {
	return ListWith(c, ListOptions{})
}

// SplitFullName splits a domain's full name into the name and extension the
// API's `domain` object carries as separate fields. The split is on the
// first dot, not the last: OpenProvider treats a second-level country
// suffix (`co.uk`, `com.au`, and the like) as one extension, so
// "example.co.uk" is name "example", extension "co.uk", not name
// "example.co", extension "uk". ok is false when domainName has no dot to
// split on, or nothing on one side of it.
func SplitFullName(domainName string) (name, extension string, ok bool) {
	name, extension, found := strings.Cut(domainName, ".")
	if !found || name == "" || extension == "" {
		return "", "", false
	}
	return name, extension, true
}
