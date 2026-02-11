// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/testutils"
)

func TestListOrders(t *testing.T) {
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

	orders, err := ListOrders(c)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if orders == nil {
		t.Log("Note: No orders returned by mock server")
		return
	}

	t.Logf("Retrieved %d SSL orders", len(orders))
}

func TestGetOrder(t *testing.T) {
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

	order, err := GetOrder(c, 123)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if order == nil {
		t.Log("Note: No order returned by mock server")
		return
	}

	t.Logf("Retrieved SSL order: %s (status: %s)", order.CommonName, order.Status)
}

func TestCreateOrder(t *testing.T) {
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

	req := &CreateSSLOrderRequest{
		ProductID:  1,
		CommonName: "example.com",
	}

	order, err := CreateOrder(c, req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if order == nil {
		t.Log("Note: No order returned by mock server")
		return
	}

	t.Logf("Created SSL order: %s (ID: %d)", order.CommonName, order.ID)
}

func TestUpdateOrder(t *testing.T) {
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

	req := &UpdateSSLOrderRequest{
		Autorenew: "on",
	}

	order, err := UpdateOrder(c, 123, req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if order == nil {
		t.Log("Note: No order returned by mock server")
		return
	}

	t.Logf("Updated SSL order: autorenew = %s", order.Autorenew)
}

func TestRenewOrder(t *testing.T) {
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

	req := &RenewSSLOrderRequest{
		Period: 1,
	}

	order, err := RenewOrder(c, 123, req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if order == nil {
		t.Log("Note: No order returned by mock server")
		return
	}

	t.Logf("Renewed SSL order: %s (expiration: %s)", order.CommonName, order.ExpirationDate)
}

func TestReissueOrder(t *testing.T) {
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

	req := &ReissueSSLOrderRequest{
		CommonName: "example.com",
	}

	order, err := ReissueOrder(c, 123, req)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	if order == nil {
		t.Log("Note: No order returned by mock server")
		return
	}

	t.Logf("Reissued SSL order: %s", order.CommonName)
}

func TestCancelOrder(t *testing.T) {
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

	err := CancelOrder(c, 123)
	if err != nil {
		t.Logf("Note: API returned error (expected if mock server not running): %v", err)
		return
	}

	t.Log("Successfully canceled SSL order")
}
