package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
)

type CartAddCmd struct {
	SKU string `arg:"" help:"SKU ID to add."`
	Qty int    `help:"Quantity." default:"1"`
}

type CartUpdateCmd struct {
	Index int `arg:"" help:"Cart item index (from 'cart show')."`
	Qty   int `help:"New absolute quantity (0 removes the item)." required:""`
}

type CartRemoveCmd struct {
	Index int `arg:"" help:"Cart item index."`
}

type CartClearCmd struct{}
type CartShowCmd struct{}

type CartReorderCmd struct {
	OrderID string `arg:"" optional:"" help:"Order ID to reorder (default: most recent)."`
}

type CartCmd struct {
	Show    CartShowCmd    `cmd:"" default:"1" help:"Show current cart contents."`
	Add     CartAddCmd     `cmd:"" help:"Add a new SKU to the cart (errors if SKU already in cart; use 'cart update' to change quantity)."`
	Update  CartUpdateCmd  `cmd:"" help:"Set absolute quantity of a cart line item (0 removes it)."`
	Remove  CartRemoveCmd  `cmd:"" help:"Remove a cart line item (alias for 'cart update <index> --qty 0')."`
	Clear   CartClearCmd   `cmd:"" help:"Clear all items from cart."`
	Reorder CartReorderCmd `cmd:"" help:"Re-add items from a previous order."`
}

func (c *CartShowCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	of, err := client.GetOrderForm(g.SessionOrderFormID(client))
	if err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(of)
	}

	if len(of.Items) == 0 {
		return errfmt.New(errfmt.ExitEmpty, "cart is empty")
	}

	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	for _, t := range of.Totalizers {
		fmt.Printf("%-55s R$%.2f\n", t.Name, float64(t.Value)/100)
	}
	return nil
}

func (c *CartAddCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	orderFormID := g.SessionOrderFormID(client)

	// Fail loud if SKU already in cart — VTEX silently no-ops in that case
	// and the user gets a misleading "Added to cart" message. Use cart update
	// to change the quantity of an existing line item.
	of, err := client.GetOrderForm(orderFormID)
	if err != nil {
		return err
	}
	for i, item := range of.Items {
		if item.ID == c.SKU {
			return errfmt.Usage(fmt.Sprintf(
				"SKU %s already in cart at index %d (qty=%d); use `zonasul cart update %d --qty %d` to change quantity",
				c.SKU, i, item.Quantity, i, item.Quantity+c.Qty,
			))
		}
	}

	of, err = client.AddToCart(orderFormID, c.SKU, c.Qty)
	if err != nil {
		return err
	}
	g.PersistOrderFormID(of.OrderFormID)

	if g.CLI.JSON {
		return g.Formatter().Print(of)
	}

	outfmt.Success("Added %s to cart.", c.SKU)
	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	return nil
}

func (c *CartUpdateCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	of, err := client.GetOrderForm(g.SessionOrderFormID(client))
	if err != nil {
		return err
	}
	if len(of.Items) == 0 {
		return errfmt.Usage("cart is empty — nothing to update")
	}
	if c.Index < 0 || c.Index >= len(of.Items) {
		return errfmt.Usage(fmt.Sprintf("cart index %d out of range (0-%d) — run: zonasul cart show", c.Index, len(of.Items)-1))
	}
	if c.Qty < 0 {
		return errfmt.Usage("--qty must be >= 0 (use 0 to remove)")
	}

	prevQty := of.Items[c.Index].Quantity
	of, err = client.UpdateItemQuantity(of.OrderFormID, c.Index, c.Qty)
	if err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(of)
	}
	if c.Qty == 0 {
		outfmt.Success("Removed item %d from cart.", c.Index)
	} else {
		outfmt.Success("Updated item %d: qty %d -> %d.", c.Index, prevQty, c.Qty)
	}
	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	for _, t := range of.Totalizers {
		fmt.Printf("%-55s R$%.2f\n", t.Name, float64(t.Value)/100)
	}
	return nil
}

func (c *CartRemoveCmd) Run(g *Globals) error {
	return (&CartUpdateCmd{Index: c.Index, Qty: 0}).Run(g)
}

func (c *CartClearCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	if err := client.RemoveAllItems(g.SessionOrderFormID(client)); err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "cleared"})
	}
	outfmt.Success("Cart cleared.")
	return nil
}

func (c *CartReorderCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	orderID := c.OrderID
	if orderID == "" {
		orders, listErr := client.ListOrders()
		if listErr != nil {
			return listErr
		}
		if len(orders) == 0 {
			return errfmt.Empty()
		}
		orderID = orders[0].OrderID
		outfmt.Hint("Using most recent order: %s", orderID)
	}

	detail, err := client.GetOrder(orderID)
	if err != nil {
		return err
	}
	if len(detail.Items) == 0 {
		return errfmt.New(errfmt.ExitEmpty, "order has no items")
	}

	var lastOF *vtex.OrderForm
	for _, item := range detail.Items {
		sku := item.SKU
		if sku == "" {
			sku = item.ID
		}
		of, addErr := client.AddToCart(g.SessionOrderFormID(client), sku, item.Quantity)
		if addErr != nil {
			outfmt.Warn("Failed to add %s (%s): %s", item.Name, sku, addErr)
			continue
		}
		lastOF = of
		outfmt.Success("Added %s x%d", item.Name, item.Quantity)
	}

	if lastOF == nil {
		return errfmt.New(errfmt.ExitError, "failed to add any items")
	}

	if g.CLI.JSON {
		return g.Formatter().Print(lastOF)
	}
	fmt.Println()
	for i, item := range lastOF.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	return nil
}
