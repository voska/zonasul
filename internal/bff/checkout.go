package bff

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/vtexkit/money"
)

const CommercialPolicy = "B6A2F813-9952-4F35-FBC5-08DCF2BC3756"

const ExitSubmittedUnverified = 11

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidID(id string) error {
	if !uuid.MatchString(id) {
		return errfmt.Usage("use a BFF UUID from search; legacy numeric SKU IDs and sku.reference are not interchangeable")
	}
	return nil
}

type SKU struct {
	ID        string  `json:"id"`
	Reference string  `json:"reference"`
	Unit      int     `json:"unitOfMeasure"`
	Factor    float64 `json:"arithmeticFactor"`
}

type Product struct {
	ID              string  `json:"id"`
	SKUID           string  `json:"skuId"`
	Name            string  `json:"name"`
	SKU             SKU     `json:"sku"`
	Price           float64 `json:"salePrice"`
	ReferencePrice  float64 `json:"referencePrice"`
	PackPrice       float64 `json:"arithmeticFactorPrice"`
	QuantityOrdered float64 `json:"orderedQuantity"`
	LineTotal       float64 `json:"totalPerProduct"`
	StockAvailable  bool    `json:"isStockAvailable"`
	SKUAvailable    bool    `json:"isSkuAvailable"`
}

func (p Product) Quantity(packs int) (float64, error) {
	if packs < 0 {
		return 0, errfmt.Usage("quantity cannot be negative")
	}
	if p.SKU.Unit != 1 && p.SKU.Unit != 2 {
		return 0, errfmt.Domain("unknown BFF unit of measure")
	}
	factor := 1.0
	if p.SKU.Unit == 2 {
		factor = p.SKU.Factor
		if factor <= 0 || math.IsNaN(factor) || math.IsInf(factor, 0) {
			return 0, errfmt.Domain("missing BFF pack weight")
		}
	}
	return math.Round(float64(packs)*factor*1e6) / 1e6, nil
}

type ShippingPolicy struct {
	ID        string `json:"id"`
	Scheduled *struct {
		Minimum float64 `json:"minimumPurchaseValue"`
	} `json:"scheduledDelivery"`
}

type Schedule struct {
	Date  string `json:"scheduledDate"`
	Start string `json:"scheduledStartTime"`
	End   string `json:"scheduledEndTime"`
}

type Order struct {
	ID             string         `json:"id"`
	Status         int            `json:"status"`
	AddressID      string         `json:"orderAddressId"`
	Items          []Product      `json:"items"`
	ShippingPolicy ShippingPolicy `json:"shippingPolicy"`
	Subtotal       struct {
		Items   float64 `json:"subtotal"`
		Freight float64 `json:"frete"`
		Total   float64 `json:"total"`
	} `json:"subtotal"`
	Schedule *Schedule `json:"acceptedOrderDetails"`
}

func (c *Client) Cart(token string) (*Order, error) {
	var raw json.RawMessage
	if err := c.Request(http.MethodGet, "/cart", token, nil, &raw); err != nil {
		return nil, err
	}
	if string(raw) == `""` {
		return nil, nil
	}
	var order Order
	if json.Unmarshal(raw, &order) != nil || order.ID == "" || order.Status != 1 {
		return nil, errfmt.Domain("BFF did not return an open cart")
	}
	return &order, nil
}

func (c *Client) Order(token, id string) (*Order, error) {
	var order Order
	if err := c.Request(http.MethodGet, "/gateway/orders/"+url.PathEscape(id), token, nil, &order); err != nil {
		return nil, err
	}
	if order.ID != id || order.Status == 0 {
		return nil, errfmt.Domain("BFF order readback did not match")
	}
	return &order, nil
}

func (c *Client) Search(term string, limit int) ([]Product, error) {
	if limit < 1 || limit > 50 {
		return nil, errfmt.Usage("--limit must be between 1 and 50")
	}
	var page struct {
		Items []Product `json:"items"`
	}
	path := "/products/search?" + url.Values{"term": {term}, "page": {"1"}, "pageSize": {fmt.Sprint(limit)}}.Encode()
	if err := c.Request(http.MethodGet, path, "", nil, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

// The catalog's product ID is different from skuId. Resolve the explicit
// product handle; never reinterpret a reference number as a legacy SKU.
func (c *Client) Product(id string) (*Product, error) {
	if err := ValidID(id); err != nil {
		return nil, err
	}
	var p Product
	if err := c.Request(http.MethodGet, "/products/"+id, "", nil, &p); err != nil {
		return nil, err
	}
	if p.ID != id || ValidID(p.SKUID) != nil {
		return nil, errfmt.Domain("BFF product identity did not match")
	}
	return &p, nil
}

type CartItem struct {
	Quantity float64 `json:"quantity"`
	SKUID    string  `json:"skuId"`
	Unit     int     `json:"unitOfMeasure"`
}

func (o *Order) CartItems() []CartItem {
	items := make([]CartItem, 0, len(o.Items))
	for _, p := range o.Items {
		items = append(items, CartItem{p.QuantityOrdered, p.SKUID, p.SKU.Unit})
	}
	return items
}

func (c *Client) UpdateCart(token, id string, items []CartItem) (*Order, error) {
	if items == nil {
		items = []CartItem{}
	}
	seen := map[string]bool{}
	for _, i := range items {
		if ValidID(i.SKUID) != nil || seen[i.SKUID] || i.Quantity <= 0 || math.IsNaN(i.Quantity) || math.IsInf(i.Quantity, 0) || (i.Unit != 1 && i.Unit != 2) {
			return nil, errfmt.Usage("invalid BFF cart item")
		}
		seen[i.SKUID] = true
	}
	// Even a failed/ambiguous acknowledgement may have applied the absolute
	// list. Read state once and never invite an automatic relative-add retry.
	_ = c.Request(http.MethodPut, "/cart/update", token, map[string]any{"orderId": id, "items": items, "applyGiftcard": false}, nil)
	order, err := c.Order(token, id)
	if err != nil {
		return nil, errfmt.Domain("cart update submitted, readback failed; inspect cart before retrying")
	}
	if order.Status != 1 || len(order.Items) != len(items) {
		return nil, errfmt.Domain("cart update not verified; inspect cart before retrying")
	}
	actual := map[string]CartItem{}
	for _, i := range order.CartItems() {
		actual[i.SKUID] = i
	}
	for _, i := range items {
		a, ok := actual[i.SKUID]
		if !ok || a.Unit != i.Unit || math.Abs(a.Quantity-i.Quantity) > 1e-6 {
			return nil, errfmt.Domain("cart update not verified; inspect cart before retrying")
		}
	}
	return order, nil
}

type Address struct {
	ID             string `json:"id"`
	Default        bool   `json:"isDefault"`
	Country        string `json:"country"`
	City           string `json:"city"`
	UF             string `json:"uf"`
	Neighborhood   string `json:"neighborhood"`
	Street         string `json:"street"`
	Number         string `json:"number"`
	ZipCode        string `json:"zipCode"`
	AdditionalInfo string `json:"additionalInfo"`
	Reference      string `json:"reference"`
}

func (c *Client) CreateCart(token string) (*Order, error) {
	var addresses []Address
	if err := c.Request(http.MethodGet, "/addresses", token, nil, &addresses); err != nil {
		return nil, err
	}
	var a *Address
	for i := range addresses {
		if addresses[i].Default {
			if a != nil {
				return nil, errfmt.Config("multiple default BFF addresses")
			}
			a = &addresses[i]
		}
	}
	if a == nil {
		return nil, errfmt.Config("save a default delivery address on zonasul.com.br")
	}
	var methods struct {
		Scheduled []struct {
			ID string `json:"shippingPolicyId"`
		} `json:"scheduledDelivery"`
	}
	if err := c.Request(http.MethodGet, "/delivery-methods?zipCode="+url.QueryEscape(a.ZipCode), token, nil, &methods); err != nil {
		return nil, err
	}
	if len(methods.Scheduled) != 1 || methods.Scheduled[0].ID == "" {
		return nil, errfmt.Domain("expected one scheduled delivery policy; choose delivery on zonasul.com.br")
	}
	device := make([]byte, 16)
	if _, err := rand.Read(device); err != nil {
		return nil, errfmt.Config("could not generate cart device ID")
	}
	payload := map[string]any{"shippingPolicyId": methods.Scheduled[0].ID, "commercialPolicyId": CommercialPolicy, "deviceId": fmt.Sprintf("%x-%x-%x-%x-%x", device[:4], device[4:6], device[6:8], device[8:10], device[10:]), "origin": 2,
		"orderAddress": map[string]any{"country": a.Country, "city": a.City, "state": a.UF, "uf": a.UF, "neighborhood": a.Neighborhood, "street": a.Street, "number": a.Number, "zipCode": a.ZipCode, "reference": a.Reference, "addressType": "residence", "additionalInfo": a.AdditionalInfo}}
	var result struct {
		OrderID string `json:"orderId"`
	}
	if err := c.Request(http.MethodPost, "/regionalizations/default", token, payload, &result); err != nil {
		return nil, err
	}
	if result.OrderID == "" {
		return nil, errfmt.Domain("regionalization did not return an order ID")
	}
	payload["orderId"] = result.OrderID
	result.OrderID = ""
	if err := c.Request(http.MethodPost, "/regionalizations", token, payload, &result); err != nil {
		return nil, err
	}
	if result.OrderID == "" {
		return nil, errfmt.Domain("regionalization did not return the active order ID")
	}
	order, err := c.Order(token, result.OrderID)
	if err != nil {
		return nil, err
	}
	if order.Status != 1 || len(order.Items) != 0 {
		return nil, errfmt.Domain("regionalization did not create an empty open cart")
	}
	active, err := c.Cart(token)
	if err != nil || active == nil || active.ID != order.ID {
		return nil, errfmt.Domain("regionalization active cart did not match; inspect cart before retrying")
	}
	return order, nil
}

type Window struct {
	Index     int            `json:"index"`
	ID        string         `json:"id"`
	Date      string         `json:"date"`
	StartTime string         `json:"startTime"`
	EndTime   string         `json:"endTime"`
	Price     money.Centavos `json:"price"`
	Start     time.Time      `json:"start"`
	End       time.Time      `json:"end"`
	ListPrice money.Centavos `json:"lisPrice"`
	Tax       money.Centavos `json:"tax"`
}

func (c *Client) Windows(token, policy string) ([]Window, error) {
	if policy == "" {
		return nil, errfmt.Config("cart has no shipping policy; choose delivery on zonasul.com.br")
	}
	var days []struct {
		Date struct {
			Day   string `json:"day"`
			Month string `json:"month"`
			Year  string `json:"year"`
		} `json:"date"`
		Intervals []struct {
			ID    string  `json:"id"`
			Start string  `json:"startTime"`
			End   string  `json:"endTime"`
			Price float64 `json:"totalShippingCost"`
		} `json:"intervals"`
	}
	if err := c.Request(http.MethodGet, "/submission-windows/"+url.PathEscape(policy), token, nil, &days); err != nil {
		return nil, err
	}
	windows := []Window{}
	for _, day := range days {
		date := fmt.Sprintf("%s-%02s-%02s", day.Date.Year, day.Date.Month, day.Date.Day)
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, errfmt.Domain("BFF returned an invalid window date")
		}
		for _, i := range day.Intervals {
			if i.ID == "" || len(i.Start) < 5 || len(i.End) < 5 {
				return nil, errfmt.Domain("BFF returned an incomplete window")
			}
			start, err := time.Parse(time.RFC3339, date+"T"+clock(i.Start)+":00-03:00")
			if err != nil {
				return nil, errfmt.Domain("BFF returned an invalid window start")
			}
			end, err := time.Parse(time.RFC3339, date+"T"+clock(i.End)+":00-03:00")
			if err != nil || !end.After(start) {
				return nil, errfmt.Domain("BFF returned an invalid window end")
			}
			windows = append(windows, Window{Index: len(windows), ID: i.ID, Date: date, StartTime: i.Start, EndTime: i.End, Price: money.Centavos(math.Round(i.Price * 100)), Start: start, End: end})
		}
	}
	return windows, nil
}

func (c *Client) SetWindow(token string, order *Order, w Window) (*Order, error) {
	p := map[string]any{"orderId": order.ID, "commercialPolicyId": CommercialPolicy, "scheduledDate": w.Date, "submissionWindowId": w.ID}
	if err := c.Request(http.MethodPost, "/gateway/orders/update", token, p, nil); err != nil {
		return nil, err
	}
	updated, err := c.Order(token, order.ID)
	if err != nil {
		return nil, errfmt.Domain("window submitted, readback failed; inspect order before retrying")
	}
	s := updated.Schedule
	if updated.Status != 1 || updated.ShippingPolicy.ID != order.ShippingPolicy.ID || s == nil || !strings.HasPrefix(s.Date, w.Date) || clock(s.Start) != clock(w.StartTime) || clock(s.End) != clock(w.EndTime) {
		return nil, errfmt.Domain("delivery window readback did not match; payment was not sent")
	}
	return updated, nil
}

func clock(s string) string {
	if len(s) > 5 {
		return s[:5]
	}
	return s
}

type PaymentMethod struct {
	ID      string `json:"id"`
	Method  int    `json:"paymentMethodId"`
	Display string `json:"displayValue"`
}

type Submission struct {
	OrderID           string `json:"orderId"`
	ResponseReceived  bool   `json:"paymentResponseReceived"`
	GatewayStatusCode *int   `json:"gatewayStatusCode,omitempty"`
}

func (m *PaymentMethod) UnmarshalJSON(data []byte) error {
	var wire struct {
		ID      string      `json:"id"`
		Method  json.Number `json:"paymentMethodId"`
		Display string      `json:"displayValue"`
		Data    string      `json:"data"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.Method == "" && wire.Data != "" {
		var details struct {
			Method json.Number `json:"paymentMethodId"`
		}
		if json.Unmarshal([]byte(wire.Data), &details) == nil {
			wire.Method = details.Method
		}
	}
	method, _ := wire.Method.Int64()
	*m = PaymentMethod{ID: wire.ID, Method: int(method), Display: wire.Display}
	return nil
}

func (c *Client) Cards(token string) ([]PaymentMethod, error) {
	var methods []PaymentMethod
	if err := c.Request(http.MethodGet, "/payment-methods", token, nil, &methods); err != nil {
		return nil, err
	}
	cards := []PaymentMethod{}
	for _, m := range methods {
		if m.Method == 1 && m.ID != "" {
			cards = append(cards, m)
		}
	}
	return cards, nil
}

// SubmitCard never retries a payment. Acknowledgement and even status 300 do
// not prove issuer settlement; callers must report it as unverified.
func (c *Client) SubmitCard(token string, order *Order, card PaymentMethod, w Window) (Submission, error) {
	result := Submission{OrderID: order.ID}
	if order.Status != 1 || order.AddressID == "" || card.Method != 1 || card.ID == "" {
		return result, errfmt.Domain("cart or saved credit card is not ready")
	}
	payload := map[string]any{"paymentMethodId": 1, "userPaymentMethodId": card.ID, "additionalInformation": []any{}, "commercialPolicyId": CommercialPolicy, "orderId": order.ID, "addressId": order.AddressID, "applyGiftcard": false, "installments": 1, "origin": 2, "scheduledDate": w.Date, "submissionWindowId": w.ID}
	var payment json.RawMessage
	if err := c.Request(http.MethodPost, "/payments/card", token, payload, &payment); err != nil {
		return result, errfmt.New(ExitSubmittedUnverified, "payment attempted; do not retry checkout; inspect order "+order.ID)
	}
	result.ResponseReceived = true
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(payment, &fields)
	for _, key := range []string{"status", "statusCode"} {
		if value, ok := fields[key]; ok {
			var code int
			if json.Unmarshal(value, &code) == nil {
				result.GatewayStatusCode = &code
			}
		}
	}
	// The storefront calls completion after a successful BFF envelope. Its
	// gateway status is preserved as evidence, never interpreted as settlement.
	var completion struct {
		OrderID string `json:"newOrderId"`
	}
	if err := c.Request(http.MethodPost, "/gateway/payments/order-completion", token, map[string]any{"orderId": order.ID, "paymentMethodId": "1", "origin": 2}, &completion); err != nil || completion.OrderID == "" {
		return result, errfmt.New(ExitSubmittedUnverified, "completion unverified; do not retry checkout; inspect order "+order.ID)
	}
	result.OrderID = completion.OrderID
	return result, nil
}
