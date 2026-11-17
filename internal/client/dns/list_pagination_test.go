package dns

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// pagedAPI answers a zone or record listing from `total` entries, a page at a
// time as the limit and offset of the request ask, and counts the requests.
func pagedAPI(t *testing.T, total int) (*client.Client, *int) {
	t.Helper()
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var entries []string
		for i := offset; i < offset+limit && i < total; i++ {
			if strings.HasSuffix(r.URL.Path, "/records") {
				entries = append(entries, fmt.Sprintf(`{"name":"r%d","type":"A","value":"192.0.2.1"}`, i))
			} else {
				entries = append(entries, fmt.Sprintf(`{"name":"z%d","extension":"com"}`, i))
			}
		}
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"results":[%s],"total":%d}}`, strings.Join(entries, ","), total)
	}))
	t.Cleanup(srv.Close)
	return client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()}), &requests
}

func TestListZonesPagesThroughTheAccount(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	t.Cleanup(func() { listPageSize = old })

	c, requests := pagedAPI(t, 5)
	zones, err := ListZones(c)
	if err != nil {
		t.Fatalf("ListZones: %v", err)
	}
	if len(zones) != 5 || zones[0].Name != "z0" || zones[4].Name != "z4" {
		t.Errorf("zones: got %v", zones)
	}
	if *requests != 3 {
		t.Errorf("requests: got %d, want 3", *requests)
	}
}

func TestListRecordsPagesThroughTheZone(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	t.Cleanup(func() { listPageSize = old })

	c, requests := pagedAPI(t, 4)
	records, err := ListRecords(c, "example.com")
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 4 || records[3].Name != "r3" {
		t.Errorf("records: got %v", records)
	}
	// 4 of 4 after two pages: the second page ends the listing.
	if *requests != 2 {
		t.Errorf("requests: got %d, want 2", *requests)
	}
}

func TestListZonesOfAnEmptyAccount(t *testing.T) {
	c, requests := pagedAPI(t, 0)
	zones, err := ListZones(c)
	if err != nil {
		t.Fatalf("ListZones: %v", err)
	}
	if len(zones) != 0 || *requests != 1 {
		t.Errorf("zones: got %v after %d requests", zones, *requests)
	}
}
