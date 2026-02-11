// Package dns provides functionality for working with DNS records and zones.
package dns

import (
	"encoding/json"
)

// Record represents a DNS record.
type Record struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Value            string `json:"value"`
	TTL              int    `json:"ttl,omitempty"`
	Priority         int    `json:"prio,omitempty"`
	CreationDate     string `json:"creation_date,omitempty"`
	ModificationDate string `json:"modification_date,omitempty"`
	IP               string `json:"ip,omitempty"`
}

// Zone represents a DNS zone.
type Zone struct {
	Name             string `json:"name"`
	Extension        string `json:"extension"`
	Type             string `json:"type,omitempty"`
	CreationDate     string `json:"creation_date,omitempty"`
	ModificationDate string `json:"modification_date,omitempty"`
}

// ListRecordsResponse represents the API response for listing DNS records.
type ListRecordsResponse struct {
	Code int                      `json:"code"`
	Data ListRecordsResponseData  `json:"data"`
	Desc string                   `json:"desc"`
}

// ListRecordsResponseData contains the records list data.
type ListRecordsResponseData struct {
	Results []Record `json:"results"`
	Total   int      `json:"total"`
}

// GetRecordResponse represents the API response for getting a DNS record.
type GetRecordResponse struct {
	Code int    `json:"code"`
	Data Record `json:"data"`
}

// CreateRecordRequest represents a request to create a DNS record.
type CreateRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"prio,omitempty"`
}

// CreateRecordResponse represents the API response for creating a DNS record.
type CreateRecordResponse struct {
	Code int    `json:"code"`
	Data Record `json:"data"`
}

// UpdateRecordRequest represents a request to update a DNS record.
type UpdateRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"prio,omitempty"`
}

// UpdateRecordResponse represents the API response for updating a DNS record.
type UpdateRecordResponse struct {
	Code int    `json:"code"`
	Data Record `json:"data"`
}

// DeleteRecordResponse represents the API response for deleting a DNS record.
type DeleteRecordResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
}

// ListZonesResponse represents the API response for listing DNS zones.
type ListZonesResponse struct {
	Code int                    `json:"code"`
	Data ListZonesResponseData  `json:"data"`
	Desc string                 `json:"desc"`
}

// ListZonesResponseData contains the zones list data.
type ListZonesResponseData struct {
	Results []Zone `json:"results"`
	Total   int    `json:"total"`
}

// GetZoneResponse represents the API response for getting a DNS zone.
type GetZoneResponse struct {
	Code int  `json:"code"`
	Data Zone `json:"data"`
}

// RecordUpdates represents record updates for a zone.
type RecordUpdates struct {
	Add     []Record `json:"add,omitempty"`
	Remove  []Record `json:"remove,omitempty"`
	Replace []Record `json:"replace,omitempty"`
}

// MarshalJSON customizes JSON marshaling for RecordUpdates.
func (ru RecordUpdates) MarshalJSON() ([]byte, error) {
	type Alias RecordUpdates
	return json.Marshal(struct {
		*Alias
	}{
		Alias: (*Alias)(&ru),
	})
}
