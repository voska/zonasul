package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/voska/zonasul/internal/vtex"
)

func TestAddToCart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/checkout/pub/orderForm") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orderFormId": "abc123",
				"items":       []any{},
				"totalizers":  []any{},
			})
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, "/items") {
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			items := payload["orderItems"].([]any)
			item := items[0].(map[string]any)
			if item["id"] != "6180" {
				t.Errorf("expected SKU 6180, got %v", item["id"])
			}
			if item["seller"] != "zonasulzsa" {
				t.Errorf("expected seller zonasulzsa, got %v", item["seller"])
			}

			resp := map[string]any{
				"orderFormId": "abc123",
				"items": []map[string]any{
					{
						"id":           "6180",
						"productId":    "6196",
						"name":         "Banana Prata Orgânica 800g",
						"quantity":     1,
						"price":        1099,
						"sellingPrice": 879,
						"seller":       "1",
					},
				},
				"totalizers": []map[string]any{
					{"id": "Items", "name": "Total dos Itens", "value": 879},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	of, err := c.AddToCart("", "6180", 1)
	if err != nil {
		t.Fatal(err)
	}
	if of.OrderFormID != "abc123" {
		t.Errorf("expected orderFormId abc123, got %s", of.OrderFormID)
	}
	if len(of.Items) != 1 || of.Items[0].Name != "Banana Prata Orgânica 800g" {
		t.Errorf("unexpected items: %+v", of.Items)
	}
	if of.Items[0].SellingPrice != 879 {
		t.Errorf("expected selling price 879, got %d", of.Items[0].SellingPrice)
	}
}

func TestGetOrderForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"orderFormId": "abc123",
			"items": []map[string]any{
				{
					"id":           "6180",
					"name":         "Banana Prata Orgânica 800g",
					"quantity":     2,
					"price":        1099,
					"sellingPrice": 879,
					"seller":       "1",
				},
			},
			"totalizers": []map[string]any{
				{"id": "Items", "name": "Total dos Itens", "value": 1758},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	of, err := c.GetOrderForm("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if of.OrderFormID != "abc123" {
		t.Errorf("got orderFormId %s", of.OrderFormID)
	}
	if len(of.Items) != 1 || of.Items[0].Quantity != 2 {
		t.Errorf("unexpected items: %+v", of.Items)
	}
}

func TestUpdateItemQuantity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123/items/update" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"orderFormId": "abc123",
			"items":       []any{},
			"totalizers":  []any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	of, err := c.UpdateItemQuantity("abc123", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(of.Items) != 0 {
		t.Errorf("expected empty items after removal, got %d", len(of.Items))
	}
}

func TestRemoveAllItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123/items/removeAll" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	err := c.RemoveAllItems("abc123")
	if err != nil {
		t.Fatal(err)
	}
}
