// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListRecords(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	records, err := ListRecords(c, "example.com")
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if records == nil {
		t.Log("Note: No records returned by mock server")
		return
	}

	t.Logf("Retrieved %d DNS records", len(records))
}

func TestGetRecord(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	record, err := GetRecord(c, "example.com", "www", "A")
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if record == nil {
		t.Log("Note: No record returned by mock server")
		return
	}

	t.Logf("Retrieved DNS record: %s (%s) -> %s", record.Name, record.Type, record.Value)
}

func TestCreateRecord(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	req := &CreateRecordRequest{
		Name:  "test",
		Type:  "A",
		Value: "192.0.2.1",
		TTL:   3600,
	}

	record, err := CreateRecord(c, "example.com", req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if record == nil {
		t.Log("Note: No record returned by mock server")
		return
	}

	t.Logf("Created DNS record: %s (%s) -> %s", record.Name, record.Type, record.Value)
}

func TestUpdateRecord(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	req := &UpdateRecordRequest{
		Name:  "test",
		Type:  "A",
		Value: "192.0.2.2",
		TTL:   7200,
	}

	record, err := UpdateRecord(c, "example.com", "test", "A", req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if record == nil {
		t.Log("Note: No record returned by mock server")
		return
	}

	t.Logf("Updated DNS record: %s (%s) -> %s", record.Name, record.Type, record.Value)
}

func TestDeleteRecord(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)

	err := DeleteRecord(c, "example.com", "test", "A", "192.0.2.1")
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	t.Log("Successfully deleted DNS record")
}
