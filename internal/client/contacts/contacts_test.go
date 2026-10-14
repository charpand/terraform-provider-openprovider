package contacts_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client"
	"github.com/charpand/terraform-provider-openprovider/internal/client/contacts"
)

func testClient(server *httptest.Server) *client.Client {
	return client.NewClient(client.Config{BaseURL: server.URL, Token: "test", HTTPClient: server.Client()})
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/contacts/42" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"id":42,"username":"me","rcp_client_ip_list":{"allow":["192.0.2.1","2001:db8::/32"],"deny":["198.51.100.1"]}}}`))
	}))
	defer server.Close()

	got, err := contacts.Get(testClient(server), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 42 || got.Username != "me" {
		t.Errorf("unexpected contact %+v", got)
	}
	if got.RCPClientIPList == nil || len(got.RCPClientIPList.Allow) != 2 || got.RCPClientIPList.Deny[0] != "198.51.100.1" {
		t.Errorf("unexpected list %+v", got.RCPClientIPList)
	}
}

func TestGetWithoutList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"id":42}}`))
	}))
	defer server.Close()

	got, err := contacts.Get(testClient(server), 42)
	if err != nil || got == nil || got.RCPClientIPList != nil {
		t.Fatalf("expected a contact without list, got %+v, %v", got, err)
	}
}

func TestGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":4,"desc":"not found"}`))
	}))
	defer server.Close()

	got, err := contacts.Get(testClient(server), 42)
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil), got %+v, %v", got, err)
	}
}

func TestGetServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":10005,"desc":"Access Denied"}`))
	}))
	defer server.Close()

	if _, err := contacts.Get(testClient(server), 42); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSetRCPAllowListSendsOnlyAllow(t *testing.T) {
	for name, tc := range map[string]struct {
		in   []string
		want string
	}{
		"list":  {[]string{"192.0.2.1", "10.0.0.0/8"}, `{"rcp_client_ip_list":{"allow":["192.0.2.1","10.0.0.0/8"]}}`},
		"empty": {[]string{}, `{"rcp_client_ip_list":{"allow":[]}}`},
		"nil":   {nil, `{"rcp_client_ip_list":{"allow":[]}}`},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodPut || r.URL.Path != "/v1beta/contacts/7" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if string(body) != tc.want {
					t.Errorf("body = %s, want %s", body, tc.want)
				}
				_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
			}))
			defer server.Close()

			if err := contacts.SetRCPAllowList(testClient(server), 7, tc.in); err != nil {
				t.Fatalf("SetRCPAllowList: %v", err)
			}
		})
	}
}

func TestSetRCPAllowListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":196,"desc":"invalid ip"}`))
	}))
	defer server.Close()

	if err := contacts.SetRCPAllowList(testClient(server), 7, []string{"nope"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/v1beta/contacts" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := `{"username":"ci","password":"s3cret","email":"ci@example.com","is_active":true,"name":{"first_name":"C","last_name":"I"}}`
		if string(body) != want {
			t.Errorf("body = %s, want %s", body, want)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"id":99,"token":"abc"}}`))
	}))
	defer server.Close()

	active := true
	id, err := contacts.Create(testClient(server), &contacts.Request{
		Username: "ci", Password: "s3cret", Email: "ci@example.com", IsActive: &active,
		Name: &contacts.Name{FirstName: "C", LastName: "I"},
	})
	if err != nil || id != 99 {
		t.Fatalf("Create = %d, %v", id, err)
	}
}

func TestCreateWithoutIDFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	if _, err := contacts.Create(testClient(server), &contacts.Request{Username: "x"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestUpdateSendsOnlySetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPut || r.URL.Path != "/v1beta/contacts/5" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if string(body) != `{"email":"new@example.com"}` {
			t.Errorf("body = %s", body)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
	}))
	defer server.Close()

	if err := contacts.Update(testClient(server), 5, &contacts.Request{Email: "new@example.com"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1beta/contacts/5" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"code":0,"data":{"success":true}}`))
	}))
	defer server.Close()

	if err := contacts.Delete(testClient(server), 5); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	status = http.StatusNotFound
	if err := contacts.Delete(testClient(server), 5); err != nil {
		t.Fatalf("Delete of a gone contact: %v", err)
	}
	status = http.StatusForbidden
	if err := contacts.Delete(testClient(server), 5); err == nil {
		t.Fatal("expected an error for 403")
	}
}
