package command

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/vtexkit/money"
	"github.com/voska/zonasul/internal/bff"
)

func cents(v float64) money.Centavos { return money.Centavos(math.Round(v * 100)) }

func (r *Runner) catalog(g *cli.Globals, product bool) error {
	var products []bff.Product
	if product {
		if !strings.HasPrefix(g.CLI.Product.SKU, "product:") {
			return errfmt.Usage("use product:<UUID> from search")
		}
		p, err := r.client().Product(strings.TrimPrefix(g.CLI.Product.SKU, "product:"))
		if err != nil {
			return err
		}
		products = []bff.Product{*p}
	} else {
		var err error
		products, err = r.client().Search(g.CLI.Search.Query, g.CLI.Search.Limit)
		if err != nil {
			return err
		}
	}
	if len(products) == 0 {
		return errfmt.Empty()
	}
	rows := []map[string]any{}
	for _, p := range products {
		factor, _ := p.Quantity(1)
		unit := "un"
		if p.SKU.Unit == 2 {
			unit = "kg"
		}
		rows = append(rows, map[string]any{"productId": p.ID, "sku": "product:" + p.ID, "skuId": p.SKUID, "name": p.Name, "price": cents(p.PackPrice), "listPrice": cents(p.ReferencePrice * factor), "available": nil, "availabilityVerified": false, "seller": "", "pricePerUnit": cents(p.Price), "unit": unit, "unitMultiplier": factor, "arithmeticFactor": p.SKU.Factor, "reference": p.SKU.Reference})
	}
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		if product {
			return g.Formatter().Print(rows[0])
		}
		return g.Formatter().Print(rows)
	}
	for _, p := range rows {
		fmt.Printf("%s  %-45s %s / pack\n", p["sku"], p["name"], p["price"])
	}
	return nil
}

func cartRows(order *bff.Order) ([]map[string]any, error) {
	rows := []map[string]any{}
	if order == nil {
		return rows, nil
	}
	for i, p := range order.Items {
		factor, err := p.Quantity(1)
		var packs any
		if err == nil && factor > 0 {
			q := p.QuantityOrdered / factor
			if math.Abs(q-math.Round(q)) <= 1e-5 {
				packs = int(math.Round(q))
			}
		}
		unit := "un"
		if p.SKU.Unit == 2 {
			unit = "kg"
		}
		rows = append(rows, map[string]any{"index": i, "id": "product:" + p.ID, "productId": p.ID, "skuId": p.SKUID, "name": p.Name, "quantity": packs, "quantityVerified": packs != nil, "seller": "", "measurementUnit": unit, "unitMultiplier": p.SKU.Factor, "measuredQuantity": p.QuantityOrdered, "unitOfMeasure": p.SKU.Unit, "arithmeticFactor": p.SKU.Factor, "sellingPrice": cents(p.PackPrice), "price": cents(p.PackPrice), "total": cents(p.LineTotal)})
	}
	return rows, nil
}

func printCart(g *cli.Globals, order *bff.Order) error {
	rows, err := cartRows(order)
	if err != nil {
		return err
	}
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		return g.Formatter().Print(rows)
	}
	if len(rows) == 0 {
		fmt.Println("Cart is empty.")
		return nil
	}
	for _, i := range rows {
		fmt.Printf("%-3d %-42s x%-3v %s\n", i["index"], i["name"], i["quantity"], i["total"])
	}
	fmt.Printf("Total including delivery: %s\n", cents(order.Subtotal.Total))
	return nil
}

func (r *Runner) cart(g *cli.Globals, action string) error {
	// Validate caller input before authentication or cart creation.
	add := g.CLI.Cart.Add
	if action == "add" {
		if !strings.HasPrefix(add.SKU, "product:") {
			return errfmt.Usage("use product:<UUID> from search; legacy SKU IDs cannot be used on the new storefront")
		}
		if err := bff.ValidID(strings.TrimPrefix(add.SKU, "product:")); err != nil {
			return err
		}
		if add.Qty < 1 || add.Seller != "" {
			return errfmt.Usage("--qty must be positive; --seller is not supported by BFF")
		}
	}
	if action == "update" && g.CLI.Cart.Update.Qty < 0 {
		return errfmt.Usage("--qty cannot be negative")
	}
	s, err := r.requireAuth(g)
	if err != nil {
		return err
	}
	order, err := r.client().Cart(s.Token)
	if err != nil {
		return err
	}
	if action == "show" {
		return printCart(g, order)
	}
	var added *bff.Product
	if action == "add" {
		added, err = r.client().Product(strings.TrimPrefix(add.SKU, "product:"))
		if err != nil {
			return err
		}
		if _, err = added.Quantity(add.Qty); err != nil {
			return err
		}
	}
	dry := false
	switch action {
	case "add":
		dry = add.DryRun
	case "update":
		dry = g.CLI.Cart.Update.DryRun
	case "remove":
		dry = g.CLI.Cart.Remove.DryRun
	case "clear":
		dry = g.CLI.Cart.Clear.DryRun
	}
	if order == nil {
		if action != "add" {
			return errfmt.Empty()
		}
		if dry {
			return g.Formatter().Print(map[string]any{"dryRun": true, "action": "add", "sku": add.SKU, "quantity": add.Qty, "requiresRegionalization": true})
		}
		order, err = r.client().CreateCart(s.Token)
		if err != nil {
			return err
		}
	}
	items := order.CartItems()
	switch action {
	case "add":
		p := added
		q, err := p.Quantity(add.Qty)
		if err != nil {
			return err
		}
		found := false
		for i := range items {
			if items[i].SKUID == p.SKUID {
				items[i].Quantity = math.Round((items[i].Quantity+q)*1e6) / 1e6
				found = true
			}
		}
		if !found {
			items = append(items, bff.CartItem{SKUID: p.SKUID, Quantity: q, Unit: p.SKU.Unit})
		}
	case "update", "remove":
		idx, qty := g.CLI.Cart.Update.Index, g.CLI.Cart.Update.Qty
		if action == "remove" {
			idx = g.CLI.Cart.Remove.Index
			qty = 0
		}
		if idx < 0 || idx >= len(items) {
			return errfmt.Usage("item index out of range; run cart show")
		}
		if qty == 0 {
			items = append(items[:idx], items[idx+1:]...)
		} else {
			q, err := order.Items[idx].Quantity(qty)
			if err != nil {
				return err
			}
			items[idx].Quantity = q
		}
	case "clear":
		items = []bff.CartItem{}
	}
	if dry {
		desired := *order
		desired.Items = nil
		products := map[string]bff.Product{}
		for _, p := range order.Items {
			products[p.SKUID] = p
		}
		if added != nil {
			products[added.SKUID] = *added
		}
		for _, i := range items {
			p := products[i.SKUID]
			p.QuantityOrdered = i.Quantity
			desired.Items = append(desired.Items, p)
		}
		rows, err := cartRows(&desired)
		if err != nil {
			return err
		}
		return g.Formatter().Print(map[string]any{"dryRun": true, "action": action, "orderId": order.ID, "items": rows, "request": map[string]any{"orderId": order.ID, "items": items, "applyGiftcard": false}})
	}
	updated, err := r.client().UpdateCart(s.Token, order.ID, items)
	if err != nil {
		return err
	}
	return printCart(g, updated)
}

func (r *Runner) delivery(g *cli.Globals) error {
	s, err := r.requireAuth(g)
	if err != nil {
		return err
	}
	o, err := r.client().Cart(s.Token)
	if err != nil {
		return err
	}
	if o == nil {
		return errfmt.Empty()
	}
	windows, err := r.client().Windows(s.Token, o.ShippingPolicy.ID)
	if err != nil {
		return err
	}
	if len(windows) == 0 {
		return errfmt.Domain("no delivery windows available")
	}
	limit := g.CLI.Delivery.Windows.Limit
	if limit < 0 {
		return errfmt.Usage("--limit cannot be negative")
	}
	if limit > 0 && limit < len(windows) {
		windows = windows[:limit]
	}
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		return g.Formatter().Print(windows)
	}
	for _, w := range windows {
		fmt.Printf("%-3d %s  %s–%s  %s\n", w.Index, w.Date, w.StartTime, w.EndTime, w.Price)
	}
	return nil
}

func (r *Runner) payments(g *cli.Globals) error {
	s, err := r.requireAuth(g)
	if err != nil {
		return err
	}
	cards, err := r.client().Cards(s.Token)
	if err != nil {
		return err
	}
	// Deliberately omit the server's data JSON, which contains card tokens.
	saved := []map[string]any{}
	for _, c := range cards {
		saved = append(saved, map[string]any{"id": c.ID, "paymentSystemName": "credit card", "cardNumber": masked(c.Display)})
	}
	return g.Formatter().Print(map[string]any{"accepted": []map[string]any{{"id": 1, "name": "saved credit card"}}, "savedCards": saved})
}

func masked(s string) string {
	digits := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits += string(c)
		}
	}
	if len(digits) > 4 {
		digits = digits[len(digits)-4:]
	}
	return "****" + digits
}

func checkoutBlockers(o *bff.Order, cards []bff.PaymentMethod) []string {
	blockers := []string{}
	if len(cards) == 0 {
		blockers = append(blockers, "no_saved_card: save a credit card on zonasul.com.br")
	}
	if o == nil || len(o.Items) == 0 {
		return append(blockers, "empty_cart: add items first")
	}
	if o.AddressID == "" {
		blockers = append(blockers, "no_delivery_address: choose delivery on zonasul.com.br")
	}
	minimum := 100.0
	if p := o.ShippingPolicy.Scheduled; p != nil && p.Minimum > 0 {
		minimum = p.Minimum
	}
	if o.Subtotal.Items < minimum {
		blockers = append(blockers, fmt.Sprintf("below_minimum: items must total at least %s", cents(minimum)))
	}
	for _, p := range o.Items {
		if !p.StockAvailable || !p.SKUAvailable {
			blockers = append(blockers, "item_unavailable: review cart")
			break
		}
	}
	return blockers
}

func (r *Runner) checkout(g *cli.Globals) error {
	c := g.CLI.Checkout.Run
	if c.Confirm && c.DryRun {
		return errfmt.Usage("--confirm and --dry-run cannot be combined")
	}
	if c.CVV != "" {
		return errfmt.Usage("BFF uses saved cards; do not pass --cvv")
	}
	s, err := r.requireAuth(g)
	if err != nil {
		return err
	}
	o, err := r.client().Cart(s.Token)
	if err != nil {
		return err
	}
	cards, err := r.client().Cards(s.Token)
	if err != nil {
		return err
	}
	blockers := checkoutBlockers(o, cards)
	var card *bff.PaymentMethod
	for i := range cards {
		if cards[i].ID == c.Payment {
			card = &cards[i]
		}
	}
	if len(cards) > 0 && card == nil {
		blockers = append(blockers, "select_saved_card: pass --payment <id> from checkout payments; Pix/cash unsupported")
	}
	var selected *bff.Window
	if o != nil && o.ShippingPolicy.ID != "" {
		windows, err := r.client().Windows(s.Token, o.ShippingPolicy.ID)
		if err != nil {
			return err
		}
		if c.Window >= 0 {
			if c.Window >= len(windows) {
				return errfmt.Usage("window index out of range; run delivery windows")
			}
			selected = &windows[c.Window]
		} else if o.Schedule != nil {
			for i, w := range windows {
				if strings.HasPrefix(o.Schedule.Date, w.Date) && strings.HasPrefix(o.Schedule.Start, w.StartTime[:5]) && strings.HasPrefix(o.Schedule.End, w.EndTime[:5]) {
					selected = &windows[i]
					break
				}
			}
		}
	}
	if selected == nil {
		blockers = append(blockers, "no_delivery_window: pass --window from delivery windows")
	}
	if c.Confirm {
		if len(blockers) > 0 {
			return errfmt.Domain(strings.Join(blockers, "; "))
		}
		if card == nil {
			return errfmt.Usage("pass --payment <saved-card-id> from checkout payments; Pix/cash are not supported")
		}
		if err := r.recordPaymentAttempt(g, o.ID); err != nil {
			return err
		}
		updated, err := r.client().SetWindow(s.Token, o, *selected)
		if err != nil {
			return err
		}
		if b := checkoutBlockers(updated, cards); len(b) > 0 {
			return errfmt.Domain(strings.Join(b, "; "))
		}
		submission, err := r.client().SubmitCard(s.Token, updated, *card, *selected)
		result := map[string]any{"orderId": o.ID, "cartId": o.ID, "status": "submitted_unverified", "settlementVerified": false, "total": cents(updated.Subtotal.Total), "window": selected}
		result["paymentResponseReceived"] = submission.ResponseReceived
		if submission.GatewayStatusCode != nil {
			result["gatewayStatusCode"] = *submission.GatewayStatusCode
		}
		if err != nil {
			if printErr := g.Formatter().Print(result); printErr != nil {
				return errfmt.New(bff.ExitSubmittedUnverified, "payment attempted; output failed; do not retry checkout; inspect orders "+o.ID)
			}
			return err
		}
		readback, readErr := r.client().Order(s.Token, submission.OrderID)
		if readErr == nil {
			result["statusCode"] = readback.Status
		}
		if err := g.Formatter().Print(result); err != nil {
			return errfmt.New(bff.ExitSubmittedUnverified, "payment attempted; output failed; do not retry checkout; inspect orders "+submission.OrderID)
		}
		return errfmt.New(bff.ExitSubmittedUnverified, "payment submitted, issuer settlement NOT VERIFIED; do not retry checkout; inspect orders "+submission.OrderID)
	}
	rows, err := cartRows(o)
	if err != nil {
		return err
	}
	result := map[string]any{"stage": "payment", "ready": len(blockers) == 0, "dryRun": true, "blockers": blockers, "items": rows, "window": selected, "settlementVerified": false}
	if o != nil {
		result["orderId"] = o.ID
		result["itemsTotal"] = cents(o.Subtotal.Items)
		result["total"] = cents(o.Subtotal.Total)
		if selected != nil {
			result["estimatedTotal"] = cents(o.Subtotal.Total-o.Subtotal.Freight) + selected.Price
			result["pricingNote"] = "total is the current cart readback; estimatedTotal uses the selected window and is repriced on confirmation"
		}
	}
	if err := g.Formatter().Print(result); err != nil {
		return err
	}
	if len(blockers) > 0 {
		return errfmt.Domain("checkout is not ready; see blockers")
	}
	return nil
}

func (r *Runner) orders(g *cli.Globals) error {
	if err := bff.ValidID(g.CLI.Orders.OrderID); err != nil {
		return errfmt.Usage("pass the BFF order UUID from checkout or cart; historical order listing is unavailable")
	}
	s, err := r.requireAuth(g)
	if err != nil {
		return err
	}
	o, err := r.client().Order(s.Token, g.CLI.Orders.OrderID)
	if err != nil {
		return err
	}
	rows, err := cartRows(o)
	if err != nil {
		return err
	}
	return g.Formatter().Print(map[string]any{"orderId": o.ID, "statusCode": o.Status, "value": cents(o.Subtotal.Total), "items": rows, "settlementVerified": false})
}

// Kept as a separate account read so a failed request cannot be mistaken for
// an absent address or payment instrument.
func (r *Runner) addresses(token string) ([]bff.Address, error) {
	var a []bff.Address
	err := r.client().Request(http.MethodGet, "/addresses", token, nil, &a)
	return a, err
}
