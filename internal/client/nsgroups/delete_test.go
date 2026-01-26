// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestDeleteNSGroup(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	err := nsgroups.Delete(apiClient, "test-group")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
