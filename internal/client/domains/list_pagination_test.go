package domains

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
)

// pagedAPI stands in for an account holding `total` domains, answering each
// request's `limit`/`offset` with that slice and recording every query it
// was asked with. Asking past `refuseAfter` requests answers a refused
// listing instead, to exercise a page failing partway through.
func pagedAPI(t *testing.T, total int, refuseAfter int, queries *[]string) *httptest.Server {
	t.Helper()
	all := make([]map[string]any, total)
	for i := range all {
		all[i] = map[string]any{
			"id":     i + 1,
			"domain": map[string]string{"name": "example", "extension": "com"},
			"status": "ACT",
		}
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		*queries = append(*queries, r.URL.RawQuery)

		if refuseAfter > 0 && len(*queries) > refuseAfter {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 196,
				"desc": "Authentication failure",
				"data": map[string]any{"results": []map[string]any{}, "total": 0},
			})
			return
		}

		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			limit, _ = strconv.Atoi(v)
		}
		offset := 0
		if v := r.URL.Query().Get("offset"); v != "" {
			offset, _ = strconv.Atoi(v)
		}

		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		page := []map[string]any{}
		if offset < len(all) {
			page = all[offset:end]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"results": page, "total": total},
		})
	}))
}

func TestListWithPagesThroughTheWholeAccount(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	defer func() { listPageSize = old }()

	var queries []string
	server := pagedAPI(t, 5, 0, &queries)
	defer server.Close()
	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})

	got, err := ListWith(c, ListOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected all 5 domains across pages, got %d", len(got))
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 requests of 2 to cover 5 domains, got %d: %v", len(queries), queries)
	}
}

func TestListWithHonoursAnExplicitLimit(t *testing.T) {
	var queries []string
	server := pagedAPI(t, 5, 0, &queries)
	defer server.Close()
	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})

	got, err := ListWith(c, ListOptions{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly the requested page of 2, got %d", len(got))
	}
	if len(queries) != 1 {
		t.Fatalf("an explicit page should be a single request, got %d", len(queries))
	}
}

func TestListWithRefusesAPageThatFailsPartway(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	defer func() { listPageSize = old }()

	var queries []string
	// The first page (of 2) succeeds; the second is refused.
	server := pagedAPI(t, 5, 1, &queries)
	defer server.Close()
	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})

	if _, err := ListWith(c, ListOptions{}); err == nil {
		t.Fatal("expected the refused second page to surface as an error, not a partial list")
	}
}

func TestListWithSendsTheDeletedFilterOnEachPage(t *testing.T) {
	old := listPageSize
	listPageSize = 2
	defer func() { listPageSize = old }()

	var queries []string
	server := pagedAPI(t, 3, 0, &queries)
	defer server.Close()
	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})

	isDeleted := true
	if _, err := ListWith(c, ListOptions{FullName: "example.com", IsDeleted: &isDeleted}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 requests of 2 to cover 3 domains, got %d: %v", len(queries), queries)
	}
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			t.Fatalf("parse query %q: %v", query, err)
		}
		if values.Get("is_deleted") != "true" || values.Get("full_name") != "example.com" {
			t.Errorf("each page should ask for the deleted domain, got %q", query)
		}
	}
}

func TestListWithLeavesTheDeletedFilterOut(t *testing.T) {
	var queries []string
	server := pagedAPI(t, 1, 0, &queries)
	defer server.Close()
	c := client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})

	if _, err := ListWith(c, ListOptions{FullName: "example.com"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for _, query := range queries {
		if values, _ := url.ParseQuery(query); values.Has("is_deleted") {
			t.Errorf("the default listing should send no is_deleted filter, got %q", query)
		}
	}
}
