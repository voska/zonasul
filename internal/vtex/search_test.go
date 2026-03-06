package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voska/zonasul/internal/vtex"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op := r.URL.Query().Get("operationName")
		if op != "productSearchV3" {
			t.Errorf("expected productSearchV3, got %s", op)
		}
		resp := map[string]any{
			"data": map[string]any{
				"productSearch": map[string]any{
					"products": []map[string]any{
						{
							"productId":   "6196",
							"productName": "Banana Prata Orgânica 800g",
							"items": []map[string]any{
								{
									"itemId": "6180",
									"name":   "Banana Prata Orgânica 800g",
									"sellers": []map[string]any{
										{
											"sellerId": "1",
											"commertialOffer": map[string]any{
												"Price":             10.99,
												"ListPrice":         10.99,
												"AvailableQuantity": 99999,
											},
										},
									},
									"measurementUnit": "kg",
									"unitMultiplier":  0.8,
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	results, err := c.Search("banana", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SKU != "6180" || results[0].Name != "Banana Prata Orgânica 800g" {
		t.Errorf("unexpected result: %+v", results[0])
	}
	if results[0].Price != 10.99 {
		t.Errorf("expected price 10.99, got %f", results[0].Price)
	}
}
