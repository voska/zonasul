package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
)

type CheckoutCmd struct {
	Window  int    `help:"Delivery window index." default:"-1"`
	Payment string `help:"Payment method: pix, credit, cash, vr, alelo, ticket." default:"pix" enum:"pix,credit,cash,vr,alelo,ticket"`
	CVV     string `help:"Credit card CVV for saved card payment." env:"ZONASUL_CVV"`
	Confirm bool   `help:"Actually place the order. Required safety gate."`
}

var paymentMethods = map[string]int{
	"pix":    125,
	"cash":   201,
	"vr":     401,
	"alelo":  403,
	"ticket": 404,
}

func (c *CheckoutCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	of, err := client.GetOrderForm(g.SessionOrderFormID(client))
	if err != nil {
		return err
	}

	if len(of.Items) == 0 {
		if g.CLI.JSON {
			_ = g.Formatter().Print(map[string]string{"error": "empty_cart"})
		}
		return errfmt.New(errfmt.ExitEmpty, "cart is empty — add items first")
	}

	var itemsTotal int
	for _, t := range of.Totalizers {
		if t.ID == "Items" {
			itemsTotal = t.Value
		}
	}
	if itemsTotal < 10000 {
		if g.CLI.JSON {
			_ = g.Formatter().Print(map[string]any{
				"error":    "min_order",
				"total":    itemsTotal,
				"required": 10000,
			})
		}
		return errfmt.New(errfmt.ExitMinOrder,
			fmt.Sprintf("minimum order R$100.00, current total R$%.2f", float64(itemsTotal)/100))
	}

	if err := client.SetAddress(of.OrderFormID, len(of.Items)); err != nil {
		return err
	}

	if c.Window >= 0 {
		windows, err := client.GetDeliveryWindows(of.OrderFormID)
		if err != nil {
			return err
		}
		if c.Window >= len(windows) {
			return errfmt.Usage(fmt.Sprintf("window index %d out of range (0-%d) — run: zonasul delivery windows", c.Window, len(windows)-1))
		}
		if err := client.SetShippingWindow(of.OrderFormID, windows[c.Window], len(of.Items)); err != nil {
			return err
		}
		outfmt.Hint("Delivery window set.")
	}

	of, err = client.GetOrderForm(of.OrderFormID)
	if err != nil {
		return err
	}
	var orderTotal int
	for _, t := range of.Totalizers {
		orderTotal += t.Value
	}

	savedCards, _ := client.GetSavedCards(of.OrderFormID)
	useCreditCard := c.Payment == "credit" || (c.CVV != "" && c.Payment == "pix")
	if c.CVV != "" {
		useCreditCard = true
	}

	if useCreditCard {
		card := vtex.SavedCard{
			AccountID:         "AAAA1111BBBB2222CCCC3333DDDD4444",
			PaymentSystem:     "2",
			PaymentSystemName: "Visa",
			CardNumber:        "****9999",
		}
		if len(savedCards) > 0 {
			card = savedCards[0]
		}
		if c.CVV == "" {
			return errfmt.Usage("--cvv is required for credit card payment")
		}
		outfmt.Hint("Using saved card: %s %s", card.PaymentSystemName, card.CardNumber)
		if err := client.SetPaymentWithSavedCard(of.OrderFormID, card, orderTotal); err != nil {
			return err
		}
	} else {
		paymentID := paymentMethods[c.Payment]
		outfmt.Hint("Payment: %s", c.Payment)
		if err := client.SetPayment(of.OrderFormID, paymentID, orderTotal); err != nil {
			return err
		}
	}

	if !c.Confirm {
		summary := map[string]any{
			"orderFormId": of.OrderFormID,
			"items":       of.Items,
			"totalizers":  of.Totalizers,
			"message":     "Use --confirm to place the order",
		}
		if g.CLI.JSON {
			return g.Formatter().Print(summary)
		}
		outfmt.Hint("Order summary:")
		for i, item := range of.Items {
			fmt.Printf("  %-4d %-45s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
		}
		for _, t := range of.Totalizers {
			fmt.Printf("  %-50s R$%.2f\n", t.Name, float64(t.Value)/100)
		}
		outfmt.Warn("Use --confirm to place the order.")
		return nil
	}

	tx, err := client.PlaceOrder(of.OrderFormID, orderTotal)
	if err != nil {
		return err
	}

	if useCreditCard {
		card := savedCards[0]
		outfmt.Hint("Processing credit card payment for %s %s...", card.PaymentSystemName, card.CardNumber)
		if err := client.PayWithSavedCard(tx, card, c.CVV, orderTotal); err != nil {
			return err
		}
		if err := client.GatewayCallback(tx.OrderGroup); err != nil {
			outfmt.Warn("gateway callback: %v", err)
		}
	}

	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"orderId": tx.OrderGroup, "status": "placed"})
	}
	outfmt.Success("Order placed! ID: %s", tx.OrderGroup)
	return nil
}
