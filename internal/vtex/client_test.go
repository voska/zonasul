package vtex_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattvoska/zonasul/internal/vtex"
)

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		cookie, err := r.Cookie("VtexIdclientAutCookie_zonasul")
		if err != nil || cookie.Value != "test-jwt" {
			t.Error("missing auth cookie")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	body, err := c.Get("/api/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("got %q", string(body))
	}
}

func TestClientPostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}
		w.Write([]byte(`{"created":true}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	body, err := c.PostJSON("/api/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"created":true}` {
		t.Errorf("got %q", string(body))
	}
}
