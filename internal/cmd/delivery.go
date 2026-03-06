package cmd

import (
	"fmt"

	"github.com/voska/zonasul/internal/errfmt"
)

type DeliveryWindowsCmd struct{}

type DeliveryCmd struct {
	Windows DeliveryWindowsCmd `cmd:"" help:"List available delivery windows."`
}

func (c *DeliveryWindowsCmd) Run(g *Globals) error {
	client, err := g.RequireAuth()
	if err != nil {
		return err
	}

	cfg, _ := g.LoadConfig()
	windows, err := client.GetDeliveryWindows(cfg.OrderFormID)
	if err != nil {
		return err
	}

	if len(windows) == 0 {
		if g.CLI.JSON {
			fmt.Println("[]")
		}
		return errfmt.Empty()
	}

	if g.CLI.JSON {
		return g.Formatter().Print(windows)
	}

	for i, w := range windows {
		price := "Gratis"
		if w.Price > 0 {
			price = fmt.Sprintf("R$%.2f", float64(w.Price)/100)
		}
		fmt.Printf("%-4d %s — %s  %s\n", i, w.Start.Local().Format("Mon 02 Jan 15:04"), w.End.Local().Format("15:04"), price)
	}
	return nil
}
