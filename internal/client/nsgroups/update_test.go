// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestUpdateNSGroup(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &nsgroups.UpdateNSGroupRequest{
		Nameservers: []nsgroups.Nameserver{
			{Name: "ns3.example.com"},
			{Name: "ns4.example.com"},
		},
	}

	group, err := nsgroups.Update(apiClient, "test-group", req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server (check your swagger examples)")
	}
}
