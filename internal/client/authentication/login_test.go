// Package authentication_test contains tests for the authentication package.
package authentication_test

import (
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/authentication"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestLogin(t *testing.T) {
	apiClient := testutils.SetupTestClient()

	token, err := authentication.Login(apiClient.HTTPClient, apiClient.BaseURL, "127.0.0.1", "test", "test")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == nil {
		t.Log("Note: No token returned by mock server (check your swagger examples)")
	}
}
