package bff

import (
	"encoding/json"
	"errors"
	"github.com/voska/vtexkit/cli/errfmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testSKU = "11111111-1111-4111-8111-111111111111"

func respond(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func TestCartEmptyAndInvalidResponses(t *testing.T) {
	for _, data := range []string{`""`, `null`, `{}`, `{"id":"order","status":300}`} {
		t.Run(data, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"success":true,"data":` + data + `}`))
			}))
			defer srv.Close()
			cart, err := New(srv.URL, srv.Client()).Cart("token")
			if data == `""` {
				if err != nil || cart != nil {
					t.Fatalf("cart=%v err=%v", cart, err)
				}
			} else if err == nil {
				t.Fatal("invalid/non-cart state accepted")
			}
		})
	}
}

func TestUpdateUsesAbsoluteListAndVerifiesReadback(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "matched", true: "stale"}[stale], func(t *testing.T) {
			writes, reads := 0, 0
			items := []CartItem{{SKUID: testSKU, Quantity: 0.52, Unit: 2}, {SKUID: "22222222-2222-4222-8222-222222222222", Quantity: 2, Unit: 1}}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/cart/update":
					writes++
					if r.Method != "PUT" {
						t.Error("wrong mutation method")
					}
					var payload struct {
						OrderID string     `json:"orderId"`
						Items   []CartItem `json:"items"`
						Gift    bool       `json:"applyGiftcard"`
					}
					if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.OrderID != "order" || payload.Gift || len(payload.Items) != 2 || payload.Items[0].Quantity != 0.52 {
						t.Fatal("wrong full cart payload")
					}
					respond(w, []any{map[string]any{"id": "order", "status": 1}})
				case "/gateway/orders/order":
					reads++
					qty := 0.52
					if stale {
						qty = 0.13
					}
					respond(w, map[string]any{"id": "order", "status": 1, "items": []any{
						map[string]any{"skuId": items[0].SKUID, "orderedQuantity": qty, "sku": map[string]any{"unitOfMeasure": 2}},
						map[string]any{"skuId": items[1].SKUID, "orderedQuantity": 2, "sku": map[string]any{"unitOfMeasure": 1}},
					}})
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
			}))
			defer srv.Close()
			_, err := New(srv.URL, srv.Client()).UpdateCart("token", "order", items)
			if (err != nil) != stale || writes != 1 || reads != 1 {
				t.Fatalf("writes=%d reads=%d err=%v", writes, reads, err)
			}
		})
	}
}

func TestWeightQuantityAndLegacyIDs(t *testing.T) {
	p := Product{SKUID: testSKU, SKU: SKU{ID: testSKU, Unit: 2, Factor: 0.13}}
	q, err := p.Quantity(4)
	if err != nil || q != 0.52 {
		t.Fatalf("quantity=%v err=%v", q, err)
	}
	for _, id := range []string{"9263", "../order", "uuid?token=x", ""} {
		if ValidID(id) == nil {
			t.Fatalf("accepted unsafe/legacy id %q", id)
		}
	}
	p.SKU.Factor = 0
	if _, err = p.Quantity(4); err == nil {
		t.Fatal("missing weight factor accepted")
	}
}

func TestWindowReadbackMustMatchDateAndTimes(t *testing.T) {
	for _, changed := range []bool{false, true} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				respond(w, true)
				return
			}
			end := "18:00:00"
			if changed {
				end = "20:00:00"
			}
			respond(w, map[string]any{"id": "order", "status": 1, "shippingPolicy": map[string]any{"id": "policy"}, "acceptedOrderDetails": map[string]any{"scheduledDate": "2030-01-02T00:00:00", "scheduledStartTime": "13:00:00", "scheduledEndTime": end}})
		}))
		w := Window{ID: testSKU, Date: "2030-01-02", StartTime: "13:00:00", EndTime: "18:00:00"}
		_, err := New(srv.URL, srv.Client()).SetWindow("token", &Order{ID: "order", ShippingPolicy: ShippingPolicy{ID: "policy"}}, w)
		srv.Close()
		if (err != nil) != changed {
			t.Fatalf("changed=%v err=%v", changed, err)
		}
	}
}

func TestCreateCartUsesSecondRegionalizationOrder(t *testing.T) {
	calls := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/addresses":
			respond(w, []any{map[string]any{"id": "address", "isDefault": true, "zipCode": "00000000"}})
		case "/delivery-methods":
			respond(w, map[string]any{"scheduledDelivery": []any{map[string]any{"shippingPolicyId": "policy"}}})
		case "/regionalizations/default":
			respond(w, map[string]any{"orderId": "empty-order"})
		case "/regionalizations":
			var p map[string]any
			_ = json.NewDecoder(r.Body).Decode(&p)
			if p["orderId"] != "empty-order" {
				t.Error("missing first order id")
			}
			respond(w, map[string]any{"orderId": "active-order"})
		case "/gateway/orders/active-order", "/cart":
			respond(w, map[string]any{"id": "active-order", "status": 1})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	cart, err := New(srv.URL, srv.Client()).CreateCart("token")
	if err != nil || cart.ID != "active-order" || strings.Contains(strings.Join(calls, ","), "/gateway/orders/empty-order") {
		t.Fatalf("cart=%v calls=%v err=%v", cart, calls, err)
	}
}

func TestWindowsPreserveCentavosAndStoreTimezone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[{"date":{"year":"2030","month":"1","day":"2"},"intervals":[{"id":"window","startTime":"13:00:00","endTime":"18:00:00","totalShippingCost":17.9}]}]}`))
	}))
	defer srv.Close()
	windows, err := New(srv.URL, srv.Client()).Windows("token", "policy")
	if err != nil || len(windows) != 1 {
		t.Fatalf("windows=%v err=%v", windows, err)
	}
	w := windows[0]
	if w.Price != 1790 || w.Start.Format("2006-01-02T15:04:05Z07:00") != "2030-01-02T13:00:00-03:00" || w.End.Sub(w.Start).Hours() != 5 {
		t.Fatalf("window=%+v", w)
	}
}

func TestSavedCardTypeAcceptsNumberOrStringAndNeverReturnsToken(t *testing.T) {
	for _, wire := range []string{`{"id":"card","paymentMethodId":1,"data":"private-token"}`, `{"id":"card","paymentMethodId":"1"}`, `{"id":"card","data":"{\"paymentMethodId\":1,\"token\":\"private-token\"}"}`} {
		var card PaymentMethod
		if json.Unmarshal([]byte(wire), &card) != nil || card.Method != 1 {
			t.Fatalf("could not decode saved card type: %s", wire)
		}
		encoded, _ := json.Marshal(card)
		if strings.Contains(string(encoded), "private-token") {
			t.Fatal("token leaked")
		}
	}
}

func TestPaymentResponseAndCompletionFailureRemainUnverified(t *testing.T) {
	for _, tc := range []struct {
		name, body, completion string
		wantCompletions        int
	}{
		{"missing-data", `null`, `{"newOrderId":"completed"}`, 0},
		{"unmapped-status", `{"status":"unmapped-provider-state"}`, `{"newOrderId":"completed"}`, 1},
		{"status-code", `{"status":2,"privateToken":"never-print"}`, `{"newOrderId":"completed"}`, 1},
		{"completion-failure", `{"status":2}`, `null`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			completions := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := tc.body
				if r.URL.Path == "/gateway/payments/order-completion" {
					completions++
					body = tc.completion
				}
				_, _ = w.Write([]byte(`{"success":true,"data":` + body + `}`))
			}))
			defer srv.Close()
			result, err := New(srv.URL, srv.Client()).SubmitCard("token", &Order{ID: "order", Status: 1, AddressID: "order-address"}, PaymentMethod{ID: "card", Method: 1}, Window{})
			if completions != tc.wantCompletions {
				t.Fatalf("completions=%d", completions)
			}
			if tc.name == "status-code" || tc.name == "unmapped-status" {
				if err != nil || result.OrderID != "completed" || (tc.name == "status-code" && (result.GatewayStatusCode == nil || *result.GatewayStatusCode != 2)) {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			} else {
				var typed *errfmt.Error
				if !errors.As(err, &typed) || typed.Code != 11 {
					t.Fatalf("err=%v", err)
				}
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "never-print") {
				t.Fatal("leaked payment payload")
			}
		})
	}
}
