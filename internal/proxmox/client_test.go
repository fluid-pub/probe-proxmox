package proxmox

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fluid/probes/proxmox/internal/config"
)

func TestGetClusterResources_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/cluster/resources" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "PVEAPIToken=") {
			t.Fatalf("missing PVEAPIToken: %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"node/pve","type":"node","node":"pve","status":"online"},{"type":"qemu","vmid":100,"node":"pve","name":"vm100","status":"running"}]}`))
	}))
	defer ts.Close()

	cfg := &config.ProxmoxConfig{
		APIURL:                ts.URL,
		APIToken:              "root@pam!id=secret",
		Timeout:               "5s",
		MaxRetries:            2,
		TLSInsecureSkipVerify: false,
	}

	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := c.GetClusterResources()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].ID != "node/pve" || rows[0].Type != "node" {
		t.Fatalf("first row: %+v", rows[0])
	}
	if rows[1].VMID == nil || *rows[1].VMID != 100 {
		t.Fatalf("second row vmid: %+v", rows[1])
	}
}

func TestGetClusterResources_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("no access"))
	}))
	defer ts.Close()

	cfg := &config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "x",
		Timeout:    "5s",
		MaxRetries: 1,
	}

	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.GetClusterResources()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetQEMUVms_FilteredPath(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/cluster/resources" || r.URL.RawQuery != "type=vm" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"qemu/100","type":"qemu","vmid":100,"node":"pve","name":"vm100","status":"running"}]}`))
	}))
	defer ts.Close()

	c, err := NewClient(&config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "root@pam!id=secret",
		Timeout:    "5s",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	vms, err := c.GetQEMUVms()
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) != 1 || vms[0].VMID != 100 || vms[0].Name != "vm100" {
		t.Fatalf("unexpected vms: %+v", vms)
	}
}

func TestGetQEMUVms_FallbackFullList(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/api2/json/cluster/resources" && r.URL.RawQuery == "type=vm" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors": "bad filter"}`))
			return
		}
		if r.URL.Path == "/api2/json/cluster/resources" && r.URL.RawQuery == "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"type":"qemu","vmid":7,"node":"n1","name":"q7","status":"running"},{"type":"lxc","vmid":8,"node":"n1"}]}`))
			return
		}
		t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
	}))
	defer ts.Close()

	c, err := NewClient(&config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "t",
		Timeout:    "5s",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	vms, err := c.GetQEMUVms()
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) != 1 || vms[0].VMID != 7 {
		t.Fatalf("vms=%+v calls=%d", vms, calls)
	}
}

func TestGetAccessUsers_StringList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/access/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":["root@pam","ops@pve"]}`))
	}))
	defer ts.Close()

	c, err := NewClient(&config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "t",
		Timeout:    "5s",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := c.GetAccessUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].UserID != "root@pam" || users[1].UserID != "ops@pve" {
		t.Fatalf("users=%+v", users)
	}
}

func TestGetAccessUsers_FullObjects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/access/users" || r.URL.RawQuery != "full=1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"userid":"root@pam","email":"a@b.c","enable":1,"groups":["admins"]}]}`))
	}))
	defer ts.Close()

	c, err := NewClient(&config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "t",
		Timeout:    "5s",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := c.GetAccessUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Email != "a@b.c" || users[0].Enable == nil || *users[0].Enable != 1 {
		t.Fatalf("users=%+v", users)
	}
	if len(users[0].Groups) != 1 || users[0].Groups[0] != "admins" {
		t.Fatalf("groups=%v", users[0].Groups)
	}
}

func TestGetAccessUsers_RetryWithoutFull(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery == "full=1" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == "/api2/json/access/users" && r.URL.RawQuery == "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":["x@pve"]}`))
			return
		}
		t.Fatalf("unexpected %s?%s", r.URL.Path, r.URL.RawQuery)
	}))
	defer ts.Close()

	c, err := NewClient(&config.ProxmoxConfig{
		APIURL:     ts.URL,
		APIToken:   "t",
		Timeout:    "5s",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	users, err := c.GetAccessUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].UserID != "x@pve" || calls < 2 {
		t.Fatalf("users=%+v calls=%d", users, calls)
	}
}
