package zonasul

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voska/vtexkit/vtex"
)

func TestStoreUsesVTEXBackend(t *testing.T) {
	// The public site is a Next.js frontend and no longer serves VTEX APIs.
	if got, want := Store.BaseURL, "https://zonasul.vtexcommercestable.com.br"; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
	if got := Store.AuthCookieName(); got != "VtexIdclientAutCookie_zonasul" {
		t.Errorf("auth cookie = %q; the backend must keep the same account", got)
	}
	if got := Store.KeyringService(); got != "zonasul-cli" {
		t.Errorf("keyring service = %q; existing credentials must remain usable", got)
	}
}

func TestStartAuthUsesBackendCallback(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.URL.Host, "zonasul.vtexcommercestable.com.br"; got != want {
			t.Errorf("auth host = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("callbackUrl"), "https://zonasul.vtexcommercestable.com.br/api/vtexid/oauth/finish"; got != want {
			t.Errorf("callback = %q, want %q", got, want)
		}
		if got := r.URL.Query().Get("scope"); got != "zonasul" {
			t.Errorf("scope = %q, want zonasul", got)
		}
		w := httptest.NewRecorder()
		_, _ = w.WriteString(`{"authenticationToken":"synthetic-start"}`)
		return w.Result(), nil
	})}
	if _, err := startAuth(context.Background(), client, Store.BaseURL); err != nil {
		t.Fatal(err)
	}
}

func TestBackendOrderReadPreservesSettlementEvidence(t *testing.T) {
	client := vtex.New(Store, "synthetic-jwt")
	client.HTTPClient().Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "zonasul.vtexcommercestable.com.br" || r.URL.Path != "/api/oms/user/orders/example-01" {
			t.Errorf("order read URL = %s", r.URL.Redacted())
		}
		cookie, err := r.Cookie("VtexIdclientAutCookie_zonasul")
		if err != nil || cookie.Value != "synthetic-jwt" {
			t.Errorf("order read did not send the account auth cookie: %v", err)
		}
		w := httptest.NewRecorder()
		_, _ = w.WriteString(`{"orderId":"example-01","status":"payment-approved",
			"paymentData":{"transactions":[{"payments":[{"connectorResponses":{
				"authId":"synthetic-auth","authorizationId":"synthetic-authorization","ReturnCode":"00"
			}}]}]}}`)
		return w.Result(), nil
	})
	order, err := client.GetOrder("example-01")
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the same JSON serialization used by `orders <id> --json`.
	data, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Status   string `json:"status"`
		Payments []struct {
			ConnectorResponses map[string]string `json:"connectorResponses"`
		} `json:"payments"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "payment-approved" || len(got.Payments) != 1 {
		t.Fatalf("order status/payment evidence lost: %s", data)
	}
	for key, want := range map[string]string{
		"authId": "synthetic-auth", "authorizationId": "synthetic-authorization", "ReturnCode": "00",
	} {
		if value := got.Payments[0].ConnectorResponses[key]; value != want {
			t.Errorf("%s = %q, want %q", key, value, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
