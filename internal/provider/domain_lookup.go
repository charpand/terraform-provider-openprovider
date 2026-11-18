package provider

import (
	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
)

// domainIsFree asks the registry, through the availability check, whether
// nobody holds the domain: the one answer the account's listing cannot lag on.
func domainIsFree(c *client.Client, domainName string) (bool, error) {
	result, err := domains.CheckOne(c, domainName)
	if err != nil {
		return false, err
	}
	return result.Status == "free", nil
}

// getDomainByName finds a domain by its name using the List API.
// Returns nil if the domain is not found.
func getDomainByName(c *client.Client, domainName string) (*domains.Domain, error) {
	return findDomainByName(c, domains.ListOptions{FullName: domainName})
}

// getDeletedDomainByName finds a domain that the account had and that is now
// deleted, with the `is_deleted` filter of the List API. Returns nil if the
// account has no deleted domain with this name.
func getDeletedDomainByName(c *client.Client, domainName string) (*domains.Domain, error) {
	isDeleted := true
	return findDomainByName(c, domains.ListOptions{FullName: domainName, IsDeleted: &isDeleted})
}

// findDomainByName lists the domains that opts select and returns the one
// with the full name in opts. A listing can show a deleted domain next to
// one with the same name that the account holds now, so a domain that is not
// deleted comes first.
func findDomainByName(c *client.Client, opts domains.ListOptions) (*domains.Domain, error) {
	// Filtered on the API's side, so the lookup does not depend on the
	// account fitting in one page of the listing.
	domainList, err := domains.ListWith(c, opts)
	if err != nil {
		return nil, err
	}

	// The filter matches on the name; keep only the exact one.
	var deleted *domains.Domain
	for i, domain := range domainList {
		fullName := domain.Domain.Name + "." + domain.Domain.Extension
		if fullName != opts.FullName {
			continue
		}
		if domain.Status != domainStatusDeleted {
			return &domainList[i], nil
		}
		if deleted == nil {
			deleted = &domainList[i]
		}
	}

	return deleted, nil
}
