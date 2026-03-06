package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/voska/zonasul/internal/vtex"
)

func TestSetAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{map[string]any{"id": "123"}},
				"shippingData": map[string]any{
					"selectedAddresses": []map[string]any{
						{
							"addressId":   "addr1",
							"addressType": "residential",
							"postalCode":  "22240-003",
						},
					},
					"logisticsInfo": []map[string]any{
						{
							"slas": []map[string]any{
								{"id": "Entrega Zona Sul"},
							},
						},
					},
				},
			})
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		if _, hasAddress := payload["address"]; hasAddress {
			t.Error("payload should NOT have top-level 'address' field")
		}

		clearFlag, ok := payload["clearAddressIfPostalCodeNotFound"]
		if !ok {
			t.Error("missing clearAddressIfPostalCodeNotFound")
		} else if clearFlag != false {
			t.Errorf("expected clearAddressIfPostalCodeNotFound=false, got %v", clearFlag)
		}

		selAddrs := payload["selectedAddresses"].([]any)
		if len(selAddrs) != 1 {
			t.Errorf("expected 1 selected address, got %d", len(selAddrs))
		}

		info := payload["logisticsInfo"].([]any)
		if len(info) != 1 {
			t.Errorf("expected 1 logistics info, got %d", len(info))
		}
		li := info[0].(map[string]any)
		if li["selectedSla"] != "Entrega Zona Sul" {
			t.Errorf("expected SLA 'Entrega Zona Sul', got %v", li["selectedSla"])
		}
		if _, hasDW := li["deliveryWindow"]; hasDW {
			t.Error("SetAddress should NOT include deliveryWindow")
		}

		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	err := c.SetAddress("abc123", 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetShippingWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{map[string]any{"id": "123"}},
				"shippingData": map[string]any{
					"selectedAddresses": []map[string]any{
						{
							"addressId":   "addr1",
							"addressType": "residential",
							"postalCode":  "22240-003",
						},
					},
					"logisticsInfo": []map[string]any{
						{
							"slas": []map[string]any{
								{"id": "Entrega Zona Sul"},
							},
						},
					},
				},
			})
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)

		if _, hasAddress := payload["address"]; hasAddress {
			t.Error("payload should NOT have top-level 'address' field")
		}

		clearFlag, ok := payload["clearAddressIfPostalCodeNotFound"]
		if !ok {
			t.Error("missing clearAddressIfPostalCodeNotFound")
		} else if clearFlag != false {
			t.Errorf("expected clearAddressIfPostalCodeNotFound=false, got %v", clearFlag)
		}

		info := payload["logisticsInfo"].([]any)
		if len(info) != 1 {
			t.Errorf("expected 1 logistics info, got %d", len(info))
		}
		li := info[0].(map[string]any)
		dw := li["deliveryWindow"].(map[string]any)
		if dw["startDateUtc"] != "2026-03-04T14:01:00+00:00" {
			t.Errorf("unexpected start: %v", dw["startDateUtc"])
		}
		if dw["price"].(float64) != 700 {
			t.Errorf("expected price 700, got %v", dw["price"])
		}
		if dw["lisPrice"].(float64) != 0 {
			t.Errorf("expected lisPrice 0, got %v", dw["lisPrice"])
		}
		if dw["tax"].(float64) != 0 {
			t.Errorf("expected tax 0, got %v", dw["tax"])
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	window := vtex.DeliveryWindow{
		Start:    time.Date(2026, 3, 4, 14, 1, 0, 0, time.UTC),
		End:      time.Date(2026, 3, 4, 16, 0, 59, 0, time.UTC),
		Price:    700,
		LisPrice: 0,
		Tax:      0,
		RawStart: "2026-03-04T14:01:00+00:00",
		RawEnd:   "2026-03-04T16:00:59+00:00",
	}
	err := c.SetShippingWindow("abc123", window, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetPayment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123/attachments/paymentData" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		payments := payload["payments"].([]any)
		p := payments[0].(map[string]any)
		if p["paymentSystem"].(float64) != 125 {
			t.Errorf("expected payment system 125, got %v", p["paymentSystem"])
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	err := c.SetPayment("abc123", 125, 10000)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetPaymentWithSavedCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123/attachments/paymentData" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		payments := payload["payments"].([]any)
		p := payments[0].(map[string]any)
		if p["paymentSystem"].(float64) != 2 {
			t.Errorf("expected payment system 2, got %v", p["paymentSystem"])
		}
		if p["paymentSystemName"] != "Visa" {
			t.Errorf("expected paymentSystemName Visa, got %v", p["paymentSystemName"])
		}
		if p["group"] != "creditCardPaymentGroup" {
			t.Errorf("expected group creditCardPaymentGroup, got %v", p["group"])
		}
		if p["accountId"] != "acct-123" {
			t.Errorf("expected accountId acct-123, got %v", p["accountId"])
		}
		if p["installmentsValue"].(float64) != 13048 {
			t.Errorf("expected installmentsValue 13048, got %v", p["installmentsValue"])
		}
		if p["tokenId"] != nil {
			t.Errorf("expected tokenId nil, got %v", p["tokenId"])
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	card := vtex.SavedCard{
		AccountID:         "acct-123",
		CardNumber:        "****1234",
		PaymentSystem:     "2",
		PaymentSystemName: "Visa",
	}
	err := c.SetPaymentWithSavedCard("abc123", card, 13048)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSavedCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"paymentData": map[string]any{
				"availableAccounts": []map[string]any{
					{
						"accountId":         "acct-456",
						"cardNumber":        "****5678",
						"bin":               "411111",
						"paymentSystem":     "2",
						"paymentSystemName": "Visa",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	cards, err := c.GetSavedCards("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 saved card, got %d", len(cards))
	}
	if cards[0].AccountID != "acct-456" {
		t.Errorf("expected accountId acct-456, got %s", cards[0].AccountID)
	}
	if cards[0].PaymentSystemName != "Visa" {
		t.Errorf("expected Visa, got %s", cards[0].PaymentSystemName)
	}
}

func TestPlaceOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/checkout/pub/orderForm/abc123/transaction" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"orderGroup":  "v12345",
			"receiverUri": "https://zonasul.vtexpayments.com.br/split/v12345/payments",
			"merchantTransactions": []map[string]any{
				{
					"id":            "ZONASULZSA",
					"transactionId": "tx-abc-123",
					"payments": []map[string]any{
						{"id": "pay-001"},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	tx, err := c.PlaceOrder("abc123", 13048)
	if err != nil {
		t.Fatal(err)
	}
	if tx.OrderGroup != "v12345" {
		t.Errorf("expected order group v12345, got %s", tx.OrderGroup)
	}
	if tx.TransactionID != "tx-abc-123" {
		t.Errorf("expected transactionId tx-abc-123, got %s", tx.TransactionID)
	}
	if tx.ReceiverUri != "https://zonasul.vtexpayments.com.br/split/v12345/payments" {
		t.Errorf("expected receiverUri, got %s", tx.ReceiverUri)
	}
	if tx.MerchantName != "ZONASULZSA" {
		t.Errorf("expected merchantName ZONASULZSA, got %s", tx.MerchantName)
	}
}

func TestPayWithSavedCard(t *testing.T) {
	var capturedPath string
	var capturedPayload []map[string]any
	var capturedCookie string

	gatewaySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		cookie, err := r.Cookie("VtexIdclientAutCookie_zonasul")
		if err == nil {
			capturedCookie = cookie.Value
		}

		_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer gatewaySrv.Close()

	c := vtex.NewClient("http://unused.example.com", "test-jwt")
	c.GatewayURL = gatewaySrv.URL

	tx := &vtex.TransactionResult{
		OrderGroup:    "v99999",
		TransactionID: "tx-payment-456",
		ReceiverUri:   gatewaySrv.URL + "/split/v99999/payments",
		MerchantName:  "ZONASULZSA-zonasulzsa",
	}
	card := vtex.SavedCard{
		AccountID:         "AAAA1111BBBB2222CCCC3333DDDD4444",
		CardNumber:        "****1234",
		PaymentSystem:     "2",
		PaymentSystemName: "Visa",
	}

	err := c.PayWithSavedCard(tx, card, "123", 13748)
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := "/api/payments/pub/transactions/tx-payment-456/payments"
	if capturedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, capturedPath)
	}

	if capturedCookie != "test-jwt" {
		t.Errorf("expected auth cookie test-jwt, got %s", capturedCookie)
	}

	if len(capturedPayload) != 1 {
		t.Fatalf("expected 1 payment in array, got %d", len(capturedPayload))
	}

	p := capturedPayload[0]
	if p["paymentSystem"].(float64) != 2 {
		t.Errorf("expected paymentSystem 2, got %v", p["paymentSystem"])
	}
	if p["paymentSystemName"] != "Visa" {
		t.Errorf("expected paymentSystemName Visa, got %v", p["paymentSystemName"])
	}
	if p["group"] != "creditCardPaymentGroup" {
		t.Errorf("expected group creditCardPaymentGroup, got %v", p["group"])
	}
	if p["value"].(float64) != 13748 {
		t.Errorf("expected value 13748, got %v", p["value"])
	}
	if p["accountId"] != "AAAA1111BBBB2222CCCC3333DDDD4444" {
		t.Errorf("expected accountId, got %v", p["accountId"])
	}
	if p["currencyCode"] != "BRL" {
		t.Errorf("expected currencyCode BRL, got %v", p["currencyCode"])
	}
	if p["id"] != "ZONASULZSA-zonasulzsa" {
		t.Errorf("expected id ZONASULZSA-zonasulzsa, got %v", p["id"])
	}

	fields := p["fields"].(map[string]any)
	if fields["securityCode"] != "123" {
		t.Errorf("expected securityCode 123, got %v", fields["securityCode"])
	}
	if fields["accountId"] != "AAAA1111BBBB2222CCCC3333DDDD4444" {
		t.Errorf("expected fields.accountId, got %v", fields["accountId"])
	}

	txn := p["transaction"].(map[string]any)
	if txn["id"] != "tx-payment-456" {
		t.Errorf("expected transaction.id tx-payment-456, got %v", txn["id"])
	}
	if txn["merchantName"] != "ZONASULZSA" {
		t.Errorf("expected transaction.merchantName ZONASULZSA, got %v", txn["merchantName"])
	}
}

func TestGatewayCallback(t *testing.T) {
	var capturedPath string
	var capturedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	err := c.GatewayCallback("v99999")
	if err != nil {
		t.Fatal(err)
	}

	if capturedMethod != "POST" {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedPath != "/api/checkout/pub/gatewayCallback/v99999" {
		t.Errorf("expected path /api/checkout/pub/gatewayCallback/v99999, got %s", capturedPath)
	}
}
