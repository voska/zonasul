//go:build integration

package vtex_test

import (
	"testing"

	"github.com/mattvoska/zonasul/internal/vtex"
	"github.com/zalando/go-keyring"
)

func TestLiveSearch(t *testing.T) {
	c := vtex.NewClient(vtex.BaseURL, "")
	results, err := c.Search("banana", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for banana")
	}
	t.Logf("Found %d results", len(results))
	for _, r := range results {
		t.Logf("  SKU:%s  %s  R$%.2f", r.SKU, r.Name, r.Price)
	}
}

func TestLiveAuthStatus(t *testing.T) {
	token, err := keyring.Get("zonasul-cli", "vtex-jwt")
	if err != nil || token == "" {
		t.Skip("no stored token, skipping auth status test")
	}

	c := vtex.NewClient(vtex.BaseURL, token)
	user, err := c.AuthenticatedUser()
	if err != nil {
		t.Fatalf("auth check failed (token may be expired): %v", err)
	}
	t.Logf("Authenticated as: %s", user)
}

func TestLiveGetOrderForm(t *testing.T) {
	token, err := keyring.Get("zonasul-cli", "vtex-jwt")
	if err != nil || token == "" {
		t.Skip("no stored token, skipping orderForm test")
	}

	c := vtex.NewClient(vtex.BaseURL, token)
	of, err := c.GetOrderForm("")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("OrderForm ID: %s, Items: %d", of.OrderFormID, len(of.Items))
	for _, item := range of.Items {
		t.Logf("  %s x%d  R$%.2f", item.Name, item.Quantity, float64(item.SellingPrice)/100)
	}
}
