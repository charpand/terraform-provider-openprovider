// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListRecords lists all DNS records for a zone.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/dns/zones/{name}/records
func ListRecords(c *client.Client, zoneName string) ([]Record, error) {
	path := fmt.Sprintf("/v1beta/dns/zones/%s/records", zoneName)
	httpReq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result ListRecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Results, nil
}

// GetRecord retrieves a specific DNS record from a zone.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/dns/zones/{name}/records
// Note: The API doesn't support getting a single record directly,
// so we retrieve all records and filter by name and type.
func GetRecord(c *client.Client, zoneName string, recordName string, recordType string) (*Record, error) {
	records, err := ListRecords(c, zoneName)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Name == recordName && record.Type == recordType {
			return &record, nil
		}
	}

	return nil, fmt.Errorf("record not found: %s (type: %s)", recordName, recordType)
}

// CreateRecord creates a new DNS record in a zone.
//
// Endpoint: POST https://api.openprovider.eu/v1beta/dns/zones/{name}/records
func CreateRecord(c *client.Client, zoneName string, req *CreateRecordRequest) (*Record, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/dns/zones/%s/records", zoneName)
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result CreateRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateRecord updates an existing DNS record in a zone.
//
// Endpoint: PUT https://api.openprovider.eu/v1beta/dns/zones/{name}/records
// Note: The API uses PUT to update records by filtering on name and type.
func UpdateRecord(c *client.Client, zoneName string, recordName string, recordType string, req *UpdateRecordRequest) (*Record, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1beta/dns/zones/%s/records", zoneName)
	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result UpdateRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DeleteRecord deletes a DNS record from a zone.
//
// Endpoint: DELETE https://api.openprovider.eu/v1beta/dns/zones/{name}/records
// Note: The API uses DELETE with body to specify the record to remove.
func DeleteRecord(c *client.Client, zoneName string, recordName string, recordType string, value string) error {
	req := DeleteRecordRequest{
		Name:  recordName,
		Type:  recordType,
		Value: value,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1beta/dns/zones/%s/records", zoneName)
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s", c.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return err
	}

	var result DeleteRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	return nil
}

// DeleteRecordRequest represents a request to delete a DNS record.
type DeleteRecordRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}
