package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
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

type CartCmd struct {
	Show   CartShowCmd   `cmd:"" default:"1" help:"Show current cart contents."`
	Add    CartAddCmd    `cmd:"" help:"Add item to cart."`
	Remove CartRemoveCmd `cmd:"" help:"Remove item from cart."`
	Clear  CartClearCmd  `cmd:"" help:"Clear all items from cart."`
}

func (c *CartShowCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	cfg, _ := g.LoadConfig()
	of, err := client.GetOrderForm(cfg.OrderFormID)
	if err != nil {
		return err
	}

	cfg.OrderFormID = of.OrderFormID
	_ = g.SaveConfig(cfg)

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

	cfg, _ := g.LoadConfig()
	of, err := client.AddToCart(cfg.OrderFormID, c.SKU, c.Qty)
	if err != nil {
		return err
	}

	cfg.OrderFormID = of.OrderFormID
	_ = g.SaveConfig(cfg)

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

	cfg, _ := g.LoadConfig()
	of, err := client.UpdateItemQuantity(cfg.OrderFormID, c.Index, 0)
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

	cfg, _ := g.LoadConfig()
	if err := client.RemoveAllItems(cfg.OrderFormID); err != nil {
		return err
	}

	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "cleared"})
	}
	outfmt.Success("Cart cleared.")
	return nil
}
