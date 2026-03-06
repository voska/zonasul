package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voska/zonasul/internal/vtex"
)

func TestAuthStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/vtexid/pub/authentication/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		scope := r.URL.Query().Get("scope")
		if scope != "zonasul" {
			t.Errorf("expected scope=zonasul, got %s", scope)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authenticationToken":         "test-auth-token-123",
			"showClassicAuthentication":   false,
			"showAccessKeyAuthentication": false,
		})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	resp, err := c.AuthStart()
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthenticationToken != "test-auth-token-123" {
		t.Errorf("got %q", resp.AuthenticationToken)
	}
	if resp.ShowClassicAuthentication {
		t.Error("expected showClassicAuthentication=false")
	}
}

func TestOAuthLoginURL(t *testing.T) {
	c := vtex.NewClient("https://www.zonasul.com.br", "")
	url := c.OAuthLoginURL("test-token")
	if url == "" {
		t.Error("expected non-empty URL")
	}
	if !contains(url, "test-token") {
		t.Errorf("URL should contain auth token: %s", url)
	}
	if !contains(url, "Cliente+Zona+Sul") && !contains(url, "Cliente%20Zona%20Sul") {
		t.Errorf("URL should contain provider name: %s", url)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
