package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/voska/vtexkit/cli"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/zonasul/internal/bff"
	"github.com/zalando/go-keyring"
)

const weightedCart = `{"id":"order","status":1,"orderAddressId":"address","shippingPolicy":{"id":"policy"},"subtotal":{"subtotal":120,"frete":17.9,"total":137.9},"items":[{"id":"11111111-1111-4111-8111-111111111111","skuId":"22222222-2222-4222-8222-222222222222","name":"Bread","orderedQuantity":0.52,"isStockAvailable":true,"isSkuAvailable":true,"sku":{"unitOfMeasure":2,"arithmeticFactor":0.13},"salePrice":10,"arithmeticFactorPrice":1.3,"totalPerProduct":5.2}]}`

func checkoutServer(t *testing.T, card bool) (*Runner, func()) {
	t.Helper()
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("preview/dry-run wrote %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
			return
		}
		data := `null`
		switch r.URL.Path {
		case "/users/me":
			data = `{"id":"synthetic-user","email":"test@example.invalid"}`
		case "/cart":
			data = weightedCart
		case "/addresses":
			data = `[{"id":"address","isDefault":true}]`
		case "/payment-methods":
			data = `[]`
			if card {
				data = `[{"id":"card","paymentMethodId":1,"displayValue":"****0000"}]`
			}
		case "/submission-windows/policy":
			data = `[{"date":{"year":"2030","month":"1","day":"2"},"intervals":[{"id":"window","startTime":"13:00:00","endTime":"18:00:00","totalShippingCost":17.9}]}]`
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":` + data + `}`))
	}))
	return &Runner{Client: bff.New(srv.URL, srv.Client())}, srv.Close
}

func TestCheckoutPreviewReachesPaymentWithoutWriting(t *testing.T) {
	r, close := checkoutServer(t, false)
	defer close()
	g := globals()
	g.CLI.JSON = true
	g.CLI.Checkout.Run.Window = 0
	g.CLI.Checkout.Run.DryRun = true
	var runErr error
	text := output(t, func() { _, runErr = r.Run("checkout run", g) })
	var result struct {
		Stage              string
		Ready              bool
		SettlementVerified bool
		Blockers           []string
	}
	if json.Unmarshal([]byte(text), &result) != nil || result.Stage != "payment" || result.Ready || runErr == nil || len(result.Blockers) == 0 {
		t.Fatalf("preview=%s err=%v", text, runErr)
	}
	if !strings.Contains(text, "no_saved_card") {
		t.Fatal("missing actionable card blocker")
	}
}

func TestCartDryRunPreservesWeightAndOtherLines(t *testing.T) {
	r, close := checkoutServer(t, false)
	defer close()
	g := globals()
	g.CLI.JSON = true
	g.CLI.Cart.Update.Index = 0
	g.CLI.Cart.Update.Qty = 3
	g.CLI.Cart.Update.DryRun = true
	var runErr error
	text := output(t, func() { _, runErr = r.Run("cart update <index>", g) })
	if runErr != nil || !strings.Contains(text, `"quantity":0.39`) || !strings.Contains(text, `"quantity":3`) || !strings.Contains(text, `"dryRun":true`) {
		t.Fatalf("preview=%s err=%v", text, runErr)
	}
}

func TestDoctorPassesOnlyWithSavedCreditCard(t *testing.T) {
	for _, card := range []bool{false, true} {
		r, close := checkoutServer(t, card)
		g := globals()
		g.CLI.JSON = true
		var runErr error
		text := output(t, func() { _, runErr = r.Run("doctor", g) })
		close()
		var result struct{ Ready bool }
		if json.Unmarshal([]byte(text), &result) != nil || result.Ready != card || (runErr == nil) != card {
			t.Fatalf("card=%v report=%s err=%v", card, text, runErr)
		}
	}
}

func TestMissingCardPreventsConfirmedCheckoutWrites(t *testing.T) {
	r, close := checkoutServer(t, false)
	defer close()
	g := globals()
	g.CLI.JSON = true
	g.CLI.Checkout.Run.Window = 0
	g.CLI.Checkout.Run.Confirm = true
	g.CLI.Checkout.Run.Payment = "card"
	var err error
	_ = output(t, func() { _, err = r.Run("checkout run", g) })
	var typed *errfmt.Error
	if !errors.As(err, &typed) || typed.Code != 9 {
		t.Fatalf("err=%v", err)
	}
}

func TestConfirmedCheckoutNeverClaimsIssuerSettlement(t *testing.T) {
	for _, payFails := range []bool{false, true} {
		t.Run(fmt.Sprint(payFails), func(t *testing.T) {
			keyring.MockInit()
			_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token"}`)
			pays, completions := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data := `true`
				switch r.URL.Path {
				case "/users/me":
					data = `{"id":"synthetic-user","email":"test@example.invalid"}`
				case "/cart":
					data = weightedCart
				case "/payment-methods":
					data = `[{"id":"card","paymentMethodId":1,"displayValue":"****0000","data":"private-card-token"}]`
				case "/submission-windows/policy":
					data = `[{"date":{"year":"2030","month":"1","day":"2"},"intervals":[{"id":"window","startTime":"13:00:00","endTime":"18:00:00","totalShippingCost":17.9}]}]`
				case "/gateway/orders/update":
					if r.Method != "POST" {
						t.Error("wrong window method")
					}
				case "/gateway/orders/order", "/gateway/orders/completed-order":
					var order map[string]any
					_ = json.Unmarshal([]byte(weightedCart), &order)
					order["acceptedOrderDetails"] = map[string]any{"scheduledDate": "2030-01-02", "scheduledStartTime": "13:00:00", "scheduledEndTime": "18:00:00"}
					if pays > 0 {
						order["status"] = 300
						order["id"] = "completed-order"
					}
					encoded, _ := json.Marshal(order)
					data = string(encoded)
				case "/payments/card":
					pays++
					var body map[string]any
					if json.NewDecoder(r.Body).Decode(&body) != nil || body["userPaymentMethodId"] != "card" || body["paymentMethodId"] != float64(1) || body["orderId"] != "order" || body["addressId"] != "address" {
						t.Error("wrong payment payload")
					}
					if payFails {
						w.WriteHeader(503)
						_, _ = w.Write([]byte("private-server-body"))
						return
					}
				case "/gateway/payments/order-completion":
					completions++
					data = `{"newOrderId":"completed-order"}`
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"success":true,"data":` + data + `}`))
			}))
			defer srv.Close()
			g := globals()
			g.CLI.JSON = true
			g.CLI.Checkout.Run.Window = 0
			g.CLI.Checkout.Run.Confirm = true
			g.CLI.Checkout.Run.Payment = "card"
			var err error
			text := output(t, func() { _, err = (&Runner{Client: bff.New(srv.URL, srv.Client())}).Run("checkout run", g) })
			var typed *errfmt.Error
			if !errors.As(err, &typed) || typed.Code != 11 || pays != 1 {
				t.Fatalf("pays=%d err=%v", pays, err)
			}
			if payFails && completions != 0 || !payFails && completions != 1 {
				t.Fatalf("completions=%d", completions)
			}
			if !strings.Contains(text, `"status":"submitted_unverified"`) || !strings.Contains(text, `"settlementVerified":false`) {
				t.Fatalf("false success: %s", text)
			}
			if !payFails && !strings.Contains(text, `"orderId":"completed-order"`) {
				t.Fatal("discarded completion order id")
			}
			if strings.Contains(text+err.Error(), "private-") {
				t.Fatal("private data in output")
			}
		})
	}
}

func TestLegacyNumericCartIDRejectedBeforeNetwork(t *testing.T) {
	g := globals()
	g.CLI.Cart.Add.SKU = "9263"
	g.CLI.Cart.Add.Qty = 1
	_, err := (&Runner{Client: bff.New("http://127.0.0.1:1", nil)}).Run("cart add <sku>", g)
	var typed *errfmt.Error
	if !errors.As(err, &typed) || typed.Code != 2 {
		t.Fatalf("err=%v", err)
	}
}

func TestActualKongPathsReachBFF(t *testing.T) {
	for _, args := range [][]string{{"cart"}, {"cart", "add", "product:11111111-1111-4111-8111-111111111111"}, {"checkout", "--dry-run"}, {"orders", "11111111-1111-4111-8111-111111111111"}} {
		var c cli.CLI
		parser, err := kong.New(&c, Options("test")...)
		if err != nil {
			t.Fatal(err)
		}
		ctx, err := parser.Parse(args)
		if err != nil {
			t.Fatal(err)
		}
		keyring.MockInit()
		handled, err := (&Runner{}).Run(ctx.Command(), &cli.Globals{CLI: &c, Store: globals().Store})
		var typed *errfmt.Error
		if !handled || !errors.As(err, &typed) || typed.Code != 4 {
			t.Fatalf("path=%s handled=%v err=%v", ctx.Command(), handled, err)
		}
	}
}

func TestAddResolvesProductToDistinctSKUAndVerifiesCompleteCart(t *testing.T) {
	for _, mode := range []string{"merge", "append", "new", "dry-run"} {
		t.Run(mode, func(t *testing.T) {
			keyring.MockInit()
			_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token"}`)
			var original bff.Order
			_ = json.Unmarshal([]byte(weightedCart), &original)
			bread := original.Items[0]
			cart := original
			cart.Items = append([]bff.Product{}, original.Items...)
			if mode == "append" {
				cart.Items[0].SKUID = "33333333-3333-4333-8333-333333333333"
				cart.Items[0].SKU.Unit = 1
				cart.Items[0].QuantityOrdered = 2
			}
			active := mode != "new"
			if !active {
				cart.Items = []bff.Product{}
			}
			lookedUp, writes, reads := false, 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if mode == "dry-run" && r.Method != "GET" {
					t.Errorf("dry-run made a write: %s", r.URL.Path)
				}
				var data any = true
				switch r.URL.Path {
				case "/users/me":
					data = map[string]any{"id": "synthetic-user", "email": "test@example.invalid"}
				case "/cart":
					if active {
						data = cart
					} else {
						data = ""
					}
				case "/products/11111111-1111-4111-8111-111111111111":
					lookedUp = true
					data = bread
				case "/addresses":
					data = []any{map[string]any{"id": "address", "isDefault": true}}
				case "/delivery-methods":
					data = map[string]any{"scheduledDelivery": []any{map[string]any{"shippingPolicyId": "policy"}}}
				case "/regionalizations/default":
					if !lookedUp {
						t.Error("created order before resolving product")
					}
					data = map[string]any{"orderId": "extra-empty"}
				case "/regionalizations":
					active = true
					data = map[string]any{"orderId": "order"}
				case "/cart/update":
					writes++
					var payload struct {
						Items []bff.CartItem `json:"items"`
					}
					if json.NewDecoder(r.Body).Decode(&payload) != nil {
						t.Fatal("invalid cart payload")
					}
					if mode == "append" && len(payload.Items) != 2 {
						t.Error("lost existing cart line")
					}
					products := map[string]bff.Product{bread.SKUID: bread}
					for _, p := range cart.Items {
						products[p.SKUID] = p
					}
					cart.Items = nil
					for _, i := range payload.Items {
						p := products[i.SKUID]
						p.QuantityOrdered = i.Quantity
						cart.Items = append(cart.Items, p)
					}
					data = []bff.Order{cart}
				case "/gateway/orders/order":
					reads++
					data = cart
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
			}))
			defer srv.Close()
			g := globals()
			g.CLI.JSON = true
			g.CLI.Cart.Add.SKU = "product:" + bread.ID
			g.CLI.Cart.Add.Qty = 2
			g.CLI.Cart.Add.DryRun = mode == "dry-run"
			var err error
			text := output(t, func() { _, err = (&Runner{Client: bff.New(srv.URL, srv.Client())}).Run("cart add <sku>", g) })
			if err != nil {
				t.Fatal(err)
			}
			if mode == "dry-run" {
				if writes != 0 || !strings.Contains(text, `"quantity":6`) {
					t.Fatalf("preview=%s writes=%d", text, writes)
				}
				return
			}
			if writes != 1 || reads < 1 {
				t.Fatalf("writes=%d reads=%d", writes, reads)
			}
			expected := 0.26
			if mode == "merge" {
				expected = 0.78
			}
			found := false
			for _, p := range cart.Items {
				if p.SKUID == bread.SKUID {
					found = true
					if p.QuantityOrdered != expected {
						t.Fatalf("quantity=%v expected=%v", p.QuantityOrdered, expected)
					}
				}
			}
			if !found || !strings.Contains(text, bread.SKUID) {
				t.Fatal("product id was used as sku id")
			}
		})
	}
}

func TestCatalogOutputPreservesPackPriceAndExplicitProductHandle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var o bff.Order
		_ = json.Unmarshal([]byte(weightedCart), &o)
		var data any = o.Items[0]
		if r.URL.Path == "/products/search" {
			if r.URL.Query().Get("term") != "bread" {
				t.Error("lost search query")
			}
			data = map[string]any{"items": o.Items}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	defer srv.Close()
	r := &Runner{Client: bff.New(srv.URL, srv.Client())}
	for _, product := range []bool{false, true} {
		g := globals()
		g.CLI.JSON = true
		g.CLI.Search.Query = "bread"
		g.CLI.Search.Limit = 2
		g.CLI.Product.SKU = "product:11111111-1111-4111-8111-111111111111"
		var err error
		text := output(t, func() { err = r.catalog(g, product) })
		if err != nil || !strings.Contains(text, `"price":130`) || !strings.Contains(text, `"pricePerUnit":1000`) || !strings.Contains(text, `"sku":"product:11111111-1111-4111-8111-111111111111"`) || !strings.Contains(text, `"skuId":"22222222-2222-4222-8222-222222222222"`) {
			t.Fatalf("product=%v output=%s err=%v", product, text, err)
		}
	}
}

func TestBFFDoesNotReadLegacyCVVEnvironment(t *testing.T) {
	t.Setenv("ZONASUL_CVV", "synthetic-cvv")
	var args cli.CLI
	parser, err := kong.New(&args, Options("test")...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parser.Parse([]string{"checkout", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if args.Checkout.Run.CVV != "" {
		t.Fatal("legacy CVV environment entered BFF command")
	}
}

func TestReadyPreviewAndAdditiveExitCodeTable(t *testing.T) {
	r, close := checkoutServer(t, true)
	defer close()
	g := globals()
	g.CLI.JSON = true
	g.CLI.Checkout.Run.Window = 0
	g.CLI.Checkout.Run.Payment = "card"
	g.CLI.Checkout.Run.DryRun = true
	var err error
	text := output(t, func() { _, err = r.Run("checkout run", g) })
	if err != nil || !strings.Contains(text, `"ready":true`) {
		t.Fatalf("text=%s err=%v", text, err)
	}
	text = output(t, func() { _, err = r.Run("exit-codes", g) })
	var codes []errfmt.ExitCodeEntry
	if err != nil || json.Unmarshal([]byte(text), &codes) != nil || len(codes) != 12 || !reflect.DeepEqual(codes[:11], errfmt.ExitCodeTable()) || codes[11].Code != 11 || codes[11].Transient {
		t.Fatalf("codes=%s err=%v", text, err)
	}
}
