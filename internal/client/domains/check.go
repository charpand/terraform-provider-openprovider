package domains

import (
	"fmt"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// CheckDomain names one domain to check, as its name and extension.
type CheckDomain struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

// CheckDomainsRequest is the body of a domain availability check.
type CheckDomainsRequest struct {
	Domains []CheckDomain `json:"domains"`
}

// CheckResult is the availability of one checked domain, as the registry
// reports it: status `free` means it can be registered, `active` that
// somebody holds it.
type CheckResult struct {
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	IsPremium bool   `json:"is_premium"`
}

// CheckDomainsResponse represents a response from the domain check endpoint.
type CheckDomainsResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
	Data struct {
		Results []CheckResult `json:"results"`
	} `json:"data"`
}

// Check asks whether domains are available to register.
//
// Endpoint: POST https://api.openprovider.eu/v1/domains/check
func Check(c *client.Client, domainsToCheck []CheckDomain) ([]CheckResult, error) {
	var result CheckDomainsResponse
	if err := c.JSON("POST", "/v1/domains/check", CheckDomainsRequest{Domains: domainsToCheck}, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("domain check failed with code %d: %s", result.Code, result.Desc)
	}

	return result.Data.Results, nil
}

// CheckOne asks the registry whether a single full domain name is
// available, splitting it into the name and extension the check endpoint
// wants and unwrapping the one result a single-domain request gets back.
func CheckOne(c *client.Client, domainName string) (CheckResult, error) {
	name, extension, ok := SplitFullName(domainName)
	if !ok {
		return CheckResult{}, fmt.Errorf("domain %q must be a name and an extension, like example.com", domainName)
	}

	results, err := Check(c, []CheckDomain{{Name: name, Extension: extension}})
	if err != nil {
		return CheckResult{}, err
	}
	if len(results) != 1 {
		return CheckResult{}, fmt.Errorf("expected one check result for domain %s, got %d", domainName, len(results))
	}
	return results[0], nil
}
