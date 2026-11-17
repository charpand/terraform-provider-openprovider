package ssl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

func TestListOrdersPagesThroughTheAccount(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	t.Cleanup(func() { listPageSize = old })

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var entries []string
		for i := offset; i < offset+limit && i < 5; i++ {
			entries = append(entries, fmt.Sprintf(`{"id":%d,"common_name":"c%d.example.com","status":"ACT"}`, i+1, i))
		}
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"results":[%s],"total":5}}`, strings.Join(entries, ","))
	}))
	t.Cleanup(srv.Close)
	c := client.NewClient(client.Config{BaseURL: srv.URL, Token: "test", HTTPClient: srv.Client()})

	orders, err := ListOrders(c)
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if len(orders) != 5 || orders[0].ID != 1 || orders[4].CommonName != "c4.example.com" {
		t.Errorf("orders: got %v", orders)
	}
	if requests != 3 {
		t.Errorf("requests: got %d, want 3", requests)
	}
}
