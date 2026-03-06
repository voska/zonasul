package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
)

type SearchCmd struct {
	Query string `arg:"" help:"Search query."`
	Limit int    `help:"Max results." default:"20"`
}

func (c *SearchCmd) Run(g *Globals) error {
	client := g.Client()
	results, err := client.Search(c.Query, c.Limit)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		if g.CLI.JSON {
			fmt.Println("[]")
		}
		return errfmt.Empty()
	}

	f := g.Formatter()
	if g.CLI.JSON {
		return f.Print(results)
	}

	for i, r := range results {
		stock := ""
		if r.Available <= 0 {
			stock = " [out of stock]"
		}
		fmt.Printf("%-4d %-50s SKU:%-8s R$%.2f%s\n", i+1, r.Name, r.SKU, r.Price, stock)
	}
	return nil
}
