package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/voska/zonasul/internal/vtex"
)

// TestSearchHashCurrent asserts the hardcoded persisted-query hash matches the
// value served by VTEX today. When VTEX rotates the hash, search silently
// returns "no results" because the GraphQL endpoint answers with
// PERSISTED_QUERY_NOT_FOUND and the CLI swallows it. This test runs offline
// (no live call) and pins the expected hash so a future rotation forces an
// explicit update.
//
// To re-capture the current hash, run the live network capture and update the
// constant below. See docs/zonasul-api-research.md for the procedure.
func TestSearchHashCurrent(t *testing.T) {
	const want = "b398fc0a2fd04ea5d4f7a94c732c10fb1bf64f8f9a2b31c92aee6a5e796457c9"
	// vtex.searchHash is unexported; reach it through a Search request and
	// verify the request URL contains the expected hash.
	var capturedExt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedExt = r.URL.Query().Get("extensions")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	if _, err := c.Search("banana", 1); err != nil {
		t.Fatalf("search call failed: %v", err)
	}

	decoded, err := url.QueryUnescape(capturedExt)
	if err != nil {
		t.Fatalf("decode extensions: %v", err)
	}
	var ext struct {
		PersistedQuery struct {
			SHA256Hash string `json:"sha256Hash"`
		} `json:"persistedQuery"`
	}
	if err := json.Unmarshal([]byte(decoded), &ext); err != nil {
		t.Fatalf("parse extensions: %v", err)
	}
	if ext.PersistedQuery.SHA256Hash != want {
		t.Fatalf("search persisted-query hash is stale: got %s, want %s\n\n"+
			"Re-capture from a live browser session on www.zonasul.com.br "+
			"and update the searchHash constant in internal/vtex/search.go.",
			ext.PersistedQuery.SHA256Hash, want)
	}
}

// TestSearchSurfacesPersistedQueryNotFound ensures the CLI surfaces a stale
// hash as a clear error instead of silently returning "no results". This
// guard matters because empty results are indistinguishable from a category
// that legitimately has zero matches, which masks the bug until someone
// notices.
func TestSearchSurfacesPersistedQueryNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message":    "PersistedQueryNotFound",
					"name":       "PersistedQueryNotFoundError",
					"extensions": map[string]any{"code": "PERSISTED_QUERY_NOT_FOUND"},
				},
			},
		})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	_, err := c.Search("banana", 5)
	if err == nil {
		t.Fatal("expected error for PersistedQueryNotFound, got nil")
	}
	if !strings.Contains(err.Error(), "PERSISTED_QUERY_NOT_FOUND") &&
		!strings.Contains(err.Error(), "PersistedQueryNotFound") {
		t.Fatalf("expected error to mention PERSISTED_QUERY_NOT_FOUND, got: %v", err)
	}
}

// TestSearchSurfacesGenericGraphQLError ensures any non-empty GraphQL
// errors[] is surfaced as an error rather than swallowed into empty results.
func TestSearchSurfacesGenericGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "search backend unavailable"},
			},
		})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	_, err := c.Search("banana", 5)
	if err == nil {
		t.Fatal("expected error for GraphQL errors[], got nil")
	}
	if !strings.Contains(err.Error(), "search backend unavailable") {
		t.Fatalf("expected error to contain backend message, got: %v", err)
	}
}

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
		_ = json.NewEncoder(w).Encode(resp)
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
