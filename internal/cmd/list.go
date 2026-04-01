package cmd

import (
	"fmt"
	"sort"

	"github.com/voska/zonasul/internal/config"
	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
)

type ListShowCmd struct {
	Name string `arg:"" optional:"" help:"List name (omit to show all lists)."`
}

type ListAddCmd struct {
	Name string `arg:"" help:"List name."`
	SKU  string `arg:"" help:"SKU to add."`
}

type ListRemoveCmd struct {
	Name string `arg:"" help:"List name."`
	SKU  string `arg:"" help:"SKU to remove."`
}

type ListOrderCmd struct {
	Name string `arg:"" help:"List name."`
	Qty  int    `help:"Quantity for each item." default:"1"`
}

type ListDeleteCmd struct {
	Name string `arg:"" help:"List name to delete."`
}

type ListCmd struct {
	Show   ListShowCmd   `cmd:"" help:"Show list contents (or all lists)."`
	Add    ListAddCmd    `cmd:"" help:"Add SKU to a list."`
	Remove ListRemoveCmd `cmd:"" help:"Remove SKU from a list."`
	Order  ListOrderCmd  `cmd:"" help:"Add all items in a list to cart."`
	Delete ListDeleteCmd `cmd:"" help:"Delete an entire list."`
}

func (c *ListShowCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}

	if c.Name == "" {
		if len(lists) == 0 {
			if g.CLI.JSON {
				fmt.Println("{}")
			}
			return errfmt.Empty()
		}
		if g.CLI.JSON {
			return g.Formatter().Print(lists)
		}
		names := make([]string, 0, len(lists))
		for name := range lists {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Printf("%-20s %d items\n", name, len(lists[name]))
		}
		return nil
	}

	skus, ok := lists[c.Name]
	if !ok || len(skus) == 0 {
		if g.CLI.JSON {
			fmt.Println("[]")
		}
		return errfmt.New(errfmt.ExitEmpty, fmt.Sprintf("list %q is empty or does not exist", c.Name))
	}

	if g.CLI.JSON {
		return g.Formatter().Print(skus)
	}
	for i, sku := range skus {
		fmt.Printf("%-4d SKU:%s\n", i+1, sku)
	}
	return nil
}

func (c *ListAddCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	if !lists.Add(c.Name, c.SKU) {
		outfmt.Hint("SKU %s already in list %q.", c.SKU, c.Name)
		return nil
	}
	if err := config.SaveLists(lists); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to save lists", err)
	}
	outfmt.Success("Added SKU %s to list %q.", c.SKU, c.Name)
	if g.CLI.JSON {
		return g.Formatter().Print(lists[c.Name])
	}
	return nil
}

func (c *ListRemoveCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	if !lists.Remove(c.Name, c.SKU) {
		return errfmt.NotFound(fmt.Sprintf("SKU %s not in list %q", c.SKU, c.Name))
	}
	if len(lists[c.Name]) == 0 {
		delete(lists, c.Name)
	}
	if err := config.SaveLists(lists); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to save lists", err)
	}
	outfmt.Success("Removed SKU %s from list %q.", c.SKU, c.Name)
	if g.CLI.JSON {
		return g.Formatter().Print(lists[c.Name])
	}
	return nil
}

func (c *ListDeleteCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	if _, ok := lists[c.Name]; !ok {
		return errfmt.NotFound(fmt.Sprintf("list %q does not exist", c.Name))
	}
	delete(lists, c.Name)
	if err := config.SaveLists(lists); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to save lists", err)
	}
	outfmt.Success("Deleted list %q.", c.Name)
	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "deleted"})
	}
	return nil
}

func (c *ListOrderCmd) Run(g *Globals) error {
	return orderFromList(g, c.Name, c.Qty)
}

func orderFromList(g *Globals, name string, qty int) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	skus, ok := lists[name]
	if !ok || len(skus) == 0 {
		return errfmt.New(errfmt.ExitEmpty, fmt.Sprintf("list %q is empty or does not exist", name))
	}

	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	var lastOF *vtex.OrderForm
	for _, sku := range skus {
		of, addErr := client.AddToCart(g.SessionOrderFormID(client), sku, qty)
		if addErr != nil {
			outfmt.Warn("Failed to add SKU %s: %s", sku, addErr)
			continue
		}
		lastOF = of
		outfmt.Success("Added SKU %s", sku)
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
