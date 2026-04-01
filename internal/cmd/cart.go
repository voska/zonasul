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

type CartRemoveCmd struct {
	Index int `arg:"" help:"Cart item index to remove."`
}

type CartClearCmd struct{}
type CartShowCmd struct{}

type CartReorderCmd struct {
	OrderID string `arg:"" optional:"" help:"Order ID to reorder (default: most recent)."`
}

type CartCmd struct {
	Show    CartShowCmd    `cmd:"" default:"1" help:"Show current cart contents."`
	Add     CartAddCmd     `cmd:"" help:"Add item to cart."`
	Remove  CartRemoveCmd  `cmd:"" help:"Remove item from cart."`
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

	of, err := client.AddToCart(g.SessionOrderFormID(client), c.SKU, c.Qty)
	if err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(of)
	}

	outfmt.Success("Added %s to cart.", c.SKU)
	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	return nil
}

func (c *CartRemoveCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	of, err := client.UpdateItemQuantity(g.SessionOrderFormID(client), c.Index, 0)
	if err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(of)
	}
	outfmt.Success("Removed item %d from cart.", c.Index)
	return nil
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
