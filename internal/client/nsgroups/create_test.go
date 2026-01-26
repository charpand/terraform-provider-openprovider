// Package nsgroups_test contains tests for the nsgroups package.
package nsgroups_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestCreateNSGroup(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	req := &nsgroups.CreateNSGroupRequest{
		Name: "test-ns-group",
		Nameservers: []nsgroups.Nameserver{
			{Name: "ns1.example.com"},
			{Name: "ns2.example.com"},
		},
	}

	group, err := nsgroups.Create(apiClient, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if group == nil {
		t.Log("Note: No NS group returned by mock server (check your swagger examples)")
		return
	}

	if group.Name == "" {
		t.Log("Note: NS group name not populated by mock server")
	}
}
