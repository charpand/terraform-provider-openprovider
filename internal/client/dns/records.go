// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"fmt"
	"net/url"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// ListRecords lists all DNS records for a zone, paging through the listing.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/zones/{name}/records
func ListRecords(c *client.Client, zoneName string) ([]Record, error) {
	all := make([]Record, 0)
	for offset := 0; ; {
		path := fmt.Sprintf("/v1/dns/zones/%s/records?limit=%d&offset=%d", url.PathEscape(zoneName), listPageSize, offset)
		page, total, err := doList[ListRecordsResponse, Record](c, path, func(r ListRecordsResponse) ([]Record, int) {
			return r.Data.Results, r.Data.Total
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		offset += len(page)
		if len(page) < listPageSize || offset >= total {
			return all, nil
		}
	}
}

// GetRecord retrieves a specific DNS record from a zone.
//
// Endpoint: GET https://api.openprovider.eu/v1/dns/zones/{name}/records
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

// GetRecordWithValue retrieves a DNS record from a zone like GetRecord, but
// where a zone has several records of the same name and type (for example two
// MX records), it returns the one with `value`. If none has this value, it
// returns the first record of the name and type, as GetRecord does.
func GetRecordWithValue(c *client.Client, zoneName string, recordName string, recordType string, value string) (*Record, error) {
	records, err := ListRecords(c, zoneName)
	if err != nil {
		return nil, err
	}

	var first *Record
	for i, record := range records {
		if record.Name != recordName || record.Type != recordType {
			continue
		}
		if record.Value == value {
			return &records[i], nil
		}
		if first == nil {
			first = &records[i]
		}
	}
	if first != nil {
		return first, nil
	}

	return nil, fmt.Errorf("record not found: %s (type: %s)", recordName, recordType)
}

// CreateRecord creates a new DNS record in a zone.
//
// Endpoint: POST https://api.openprovider.eu/v1/dns/zones/{name}/records
func CreateRecord(c *client.Client, zoneName string, req *CreateRecordRequest) (*Record, error) {
	var result CreateRecordResponse
	if err := c.JSON("POST", fmt.Sprintf("/v1/dns/zones/%s/records", zoneName), req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateRecord updates an existing DNS record in a zone.
//
// Endpoint: PUT https://api.openprovider.eu/v1/dns/zones/{name}/records
// Note: The API uses PUT to update records by filtering on name and type.
func UpdateRecord(c *client.Client, zoneName string, _ string, _ string, req *UpdateRecordRequest) (*Record, error) {
	var result UpdateRecordResponse
	if err := c.JSON("PUT", fmt.Sprintf("/v1/dns/zones/%s/records", zoneName), req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DeleteRecord deletes a DNS record from a zone.
//
// Endpoint: DELETE https://api.openprovider.eu/v1/dns/zones/{name}/records
// Note: The API uses DELETE with body to specify the record to remove.
func DeleteRecord(c *client.Client, zoneName string, recordName string, recordType string, value string) error {
	var result DeleteRecordResponse
	return c.JSON("DELETE", fmt.Sprintf("/v1/dns/zones/%s/records", zoneName), DeleteRecordRequest{
		Name:  recordName,
		Type:  recordType,
		Value: value,
	}, &result)
}

// DeleteRecordRequest represents a request to delete a DNS record.
type DeleteRecordRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}
