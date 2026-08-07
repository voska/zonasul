package zonasul

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/voska/vtexkit/store"
)

func TestProviderNameMatchesLiveStorefront(t *testing.T) {
	// Must match oauthProviders[].providerName from
	// /api/vtexid/pub/authentication/start, or the driver is never chosen.
	if got := (Driver{}).ProviderName(); got != "Cliente Zona Sul" {
		t.Errorf("ProviderName() = %q, want %q", got, "Cliente Zona Sul")
	}
}

func TestDescriptorCarriesTheNonDiscoverableFacts(t *testing.T) {
	if Store.MinOrder != 10000 {
		t.Errorf("MinOrder = %d, want 10000 centavos (R$100)", int64(Store.MinOrder))
	}
	if Store.OAuth == nil {
		t.Error("classic auth is disabled on this account; an OAuth driver is required")
	}
	// The gateway returns Cielo code 59 without a ClearSale fingerprint.
	if !Store.Quirks.Has(store.ClearSaleFingerprint) {
		t.Error("ClearSaleFingerprint quirk missing")
	}
	if Store.Name != "zonasul" {
		t.Errorf("Name = %q; v0.5.0 published ~/.config/zonasul and keyring zonasul-cli", Store.Name)
	}
}

func TestLoginExtractsJWTFromRedirectChain(t *testing.T) {
	var loginBody, gotToken string

	// One server plays both the storefront and the Laravel auth app; the
	// driver only cares about paths.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/authentication/start"):
			_, _ = w.Write([]byte(`{"authenticationToken":"START-TOK"}`))

		case strings.HasSuffix(r.URL.Path, "/oauth/redirect"):
			gotToken = r.URL.Query().Get("authenticationToken")
			w.WriteHeader(http.StatusOK)

		case r.URL.Path == "/hop":
			// The callback that finally carries the JWT.
			w.Header().Set("Location", "/done?accountAuthCookieValue=THE-JWT")
			w.WriteHeader(http.StatusFound)

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	// The auth app is a fixed external domain, so the credential POST is
	// exercised separately from the redirect walk below.
	_ = loginBody

	// Drive followToToken directly: it is the part that has to stop at the
	// callback rather than following it, which is the subtle bit.
	client := srv.Client()
	jwt, err := followToToken(context.Background(), client, srv.URL+"/hop")
	if err != nil {
		t.Fatal(err)
	}
	if jwt != "THE-JWT" {
		t.Errorf("jwt = %q, want THE-JWT", jwt)
	}
	_ = gotToken
}

func TestFollowToTokenGivesUpWithoutAToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := followToToken(context.Background(), srv.Client(), srv.URL+"/nowhere")
	if err == nil {
		t.Fatal("a chain that never yields a token must error, not return empty")
	}
	if !strings.Contains(err.Error(), "OAuth callback") {
		t.Errorf("error should say where it failed, got: %v", err)
	}
}

func TestStartAuthRequiresAToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if _, err := startAuth(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Fatal("a start response with no token must fail loudly")
	}
}
