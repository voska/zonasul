package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
)

type OrdersCmd struct{}

func (c *OrdersCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	orders, err := client.ListOrders()
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		if g.CLI.JSON {
			fmt.Println("[]")
		}
		return errfmt.Empty()
	}

	if g.CLI.JSON {
		return g.Formatter().Print(orders)
	}

	for _, o := range orders {
		fmt.Printf("%-20s  %-25s  R$%.2f  %s\n", o.OrderID, o.StatusDescription, float64(o.TotalValue)/100, o.CreationDate)
	}
	return nil
}
