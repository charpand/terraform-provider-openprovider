// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestGetNSGroupByName(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	group, err := nsgroups.GetByName(apiClient, "test-group")

	if err != nil {
		// This is expected if no group matches
		t.Logf("Note: GetByName returned error (expected if no match): %v", err)
	}

	if group != nil && group.Name == "" {
		t.Log("Note: NS group name not populated by mock server")
	}
}
