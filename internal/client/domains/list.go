package domains

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListOptions narrows a domain listing. The zero value lists every domain
// the account holds, paging through as many requests as that takes.
type ListOptions struct {
	// FullName keeps only the domain with this full name (name and
	// extension), so a lookup does not page through the whole account.
	FullName string
	// Limit caps the listing to one page of at most this many domains,
	// starting at Offset. Zero means "all of them": ListWith pages through
	// the account itself rather than handing back only the API's first page.
	Limit int
	// Offset is the number of domains to skip. Only meaningful with Limit
	// set; ignored (as a starting point of 0) when it is not.
	Offset int
	// IsDeleted, when set, is sent as the API's `is_deleted` filter. True
	// asks for the domains that the account had and that are now deleted,
	// for example after a transfer to a different registrar. Nil sends no
	// filter, which is the API's default listing.
	IsDeleted *bool
}

// listPageSize is how many domains one request asks for while ListWith pages
// through an unbounded listing. The API's own default page is far smaller
// than many accounts' domain count, so listing everything must ask more than
// once; this is the size of each of those requests, not a cap on the result.
// A var, as client.retryBase is, so the tests can shrink it.
var listPageSize = 100

// ListWith retrieves domains from the Openprovider API, narrowed by the
// given options. With Limit zero (the default) it returns every domain that
// matches FullName, paging through the account's full listing rather than
// just the first page the API would otherwise hand back; a caller that wants
// one specific page sets Limit and Offset itself and gets exactly that page.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/domains
func ListWith(c *client.Client, opts ListOptions) ([]Domain, error) {
	if opts.Limit > 0 {
		page, _, err := listOnePage(c, opts)
		return page, err
	}

	all := make([]Domain, 0)
	offset := opts.Offset
	for {
		page, total, err := listOnePage(c, ListOptions{FullName: opts.FullName, Limit: listPageSize, Offset: offset, IsDeleted: opts.IsDeleted})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		offset += len(page)
		// An empty page stops the loop even if total disagrees, so a bad or
		// missing total can't spin this forever.
		if len(page) == 0 || offset >= total {
			break
		}
	}
	return all, nil
}

// listOnePage asks for exactly one page and returns it along with the
// account's total count for that filter, so ListWith knows when to stop.
func listOnePage(c *client.Client, opts ListOptions) ([]Domain, int, error) {
	query := url.Values{}
	if opts.FullName != "" {
		query.Set("full_name", opts.FullName)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.IsDeleted != nil {
		query.Set("is_deleted", strconv.FormatBool(*opts.IsDeleted))
	}

	path := "/v1beta/domains"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var results ListDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, 0, err
	}
	// The API answers a refused listing with a 200 and a non-zero `code`, and
	// no results. Told apart from an account that holds none of the names
	// asked for: a caller that reads an empty listing as "not held" would
	// otherwise drop a held domain.
	if results.Code != 0 {
		return nil, 0, fmt.Errorf("domain listing failed with code %d: %s", results.Code, results.Desc)
	}
	return results.Data.Results, results.Data.Total, nil
}
