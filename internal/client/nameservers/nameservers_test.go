package nameservers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/nameservers"
)

func testClient(t *testing.T, server *httptest.Server) *client.Client {
	t.Helper()
	return client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
}

func TestGetReadsANameserver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/dns/nameservers/ns1.example.com" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"name":"ns1.example.com","ip":"192.0.2.1","ip6":"2001:db8::1"}}`))
	}))
	defer server.Close()

	ns, err := nameservers.Get(testClient(t, server), "ns1.example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ns.IP != "192.0.2.1" || ns.IP6 != "2001:db8::1" {
		t.Errorf("expected both addresses read back, got %+v", ns)
	}
}

func TestGetReportsErrNotFoundOn404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := nameservers.Get(testClient(t, server), "ns1.example.com"); err != nameservers.ErrNotFound {
		t.Fatalf("expected ErrNotFound on a 404, got %v", err)
	}
}

func TestGetReportsErrNotFoundOnCode399(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":399,"desc":"Nameserver not found"}`))
	}))
	defer server.Close()

	if _, err := nameservers.Get(testClient(t, server), "ns1.example.com"); err != nameservers.ErrNotFound {
		t.Fatalf("expected ErrNotFound on code 399, got %v", err)
	}
}

func TestGetReportsErrNotFoundOnAnEmptyObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	if _, err := nameservers.Get(testClient(t, server), "ns1.example.com"); err != nameservers.ErrNotFound {
		t.Fatalf("expected ErrNotFound on a 200 with an empty object, got %v", err)
	}
}

func TestGetReportsAnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":61,"desc":"Authentication failure"}`))
	}))
	defer server.Close()

	if _, err := nameservers.Get(testClient(t, server), "ns1.example.com"); err == nil || err == nameservers.ErrNotFound {
		t.Fatalf("expected a non-399 error code to be a plain error, got %v", err)
	}
}

func TestCreatePostsTheNameserver(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1beta/dns/nameservers" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	err := nameservers.Create(testClient(t, server), &nameservers.Nameserver{Name: "ns1.example.com", IP: "192.0.2.1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotBody != `{"name":"ns1.example.com","ip":"192.0.2.1"}` {
		t.Errorf("unexpected request body %q", gotBody)
	}
}

func TestUpdatePutsToTheHostPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1beta/dns/nameservers/ns1.example.com" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	err := nameservers.Update(testClient(t, server), "ns1.example.com", &nameservers.Nameserver{Name: "ns1.example.com", IP: "192.0.2.2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteIsNotAnErrorWhenAlreadyGone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if err := nameservers.Delete(testClient(t, server), "ns1.example.com"); err != nil {
		t.Fatalf("expected deleting an already-gone record to be a no-op, got %v", err)
	}
}

func TestDeleteReportsAnOtherError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":61,"desc":"Authentication failure"}`))
	}))
	defer server.Close()

	if err := nameservers.Delete(testClient(t, server), "ns1.example.com"); err == nil {
		t.Fatal("expected a non-not-found error to be reported")
	}
}
