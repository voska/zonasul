package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/config"
	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
)

const favListName = "favorites"

type FavShowCmd struct{}

type FavAddCmd struct {
	SKU string `arg:"" help:"SKU to add to favorites."`
}

type FavRemoveCmd struct {
	SKU string `arg:"" help:"SKU to remove from favorites."`
}

type FavOrderCmd struct {
	Qty int `help:"Quantity for each item." default:"1"`
}

type FavCmd struct {
	Show   FavShowCmd   `cmd:"" default:"1" help:"Show favorites."`
	Add    FavAddCmd    `cmd:"" help:"Add SKU to favorites."`
	Remove FavRemoveCmd `cmd:"" help:"Remove SKU from favorites."`
	Order  FavOrderCmd  `cmd:"" help:"Add all favorites to cart."`
}

func (c *FavShowCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	skus := lists[favListName]
	if len(skus) == 0 {
		if g.CLI.JSON {
			fmt.Println("[]")
		}
		return errfmt.New(errfmt.ExitEmpty, "no favorites")
	}
	if g.CLI.JSON {
		return g.Formatter().Print(skus)
	}
	for i, sku := range skus {
		fmt.Printf("%-4d SKU:%s\n", i+1, sku)
	}
	return nil
}

func (c *FavAddCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	if !lists.Add(favListName, c.SKU) {
		outfmt.Hint("SKU %s already in favorites.", c.SKU)
		return nil
	}
	if err := config.SaveLists(lists); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to save lists", err)
	}
	outfmt.Success("Added SKU %s to favorites.", c.SKU)
	if g.CLI.JSON {
		return g.Formatter().Print(lists[favListName])
	}
	return nil
}

func (c *FavRemoveCmd) Run(g *Globals) error {
	lists, err := config.LoadLists()
	if err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to load lists", err)
	}
	if !lists.Remove(favListName, c.SKU) {
		return errfmt.NotFound(fmt.Sprintf("SKU %s not in favorites", c.SKU))
	}
	if err := config.SaveLists(lists); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to save lists", err)
	}
	outfmt.Success("Removed SKU %s from favorites.", c.SKU)
	if g.CLI.JSON {
		return g.Formatter().Print(lists[favListName])
	}
	return nil
}

func (c *FavOrderCmd) Run(g *Globals) error {
	return orderFromList(g, favListName, c.Qty)
}
