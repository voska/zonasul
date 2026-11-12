package command

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/cli/errfmt"
)

type check struct {
	Name   string `json:"check"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

func (r *Runner) doctor(g *cli.Globals) error {
	checks := []check{}
	session, err := r.requireAuth(g)
	if err != nil {
		checks = append(checks, check{Name: "logged in", Detail: err.Error(), Fix: "run: zonasul auth login"})
	} else {
		checks = append(checks, check{Name: "logged in", OK: true, Detail: "BFF identity verified"})
		for _, item := range []struct{ name, path, missing string }{
			{"delivery address", "/addresses", "restore a delivery address on www.zonasul.com.br"},
			{"saved payment methods", "/payment-methods", "restore a payment method on www.zonasul.com.br"},
		} {
			var records []json.RawMessage
			err := r.client().Request(http.MethodGet, item.path, session.Token, nil, &records)
			result := check{Name: item.name, OK: err == nil && len(records) > 0, Detail: fmt.Sprintf("%d on the BFF account", len(records))}
			if err != nil {
				result.Detail = err.Error()
			}
			if err == nil && len(records) == 0 {
				result.Fix = item.missing
			}
			checks = append(checks, result)
		}
	}
	checks = append(checks, check{Name: "CLI checkout", Detail: "BFF checkout integration is not available yet"})
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		if err := g.Formatter().Print(map[string]any{"store": "zonasul", "backend": "bff", "ready": false, "checks": checks}); err != nil {
			return err
		}
	} else {
		for _, c := range checks {
			mark := "FAIL"
			if c.OK {
				mark = "ok"
			}
			fmt.Printf("%-5s %-20s %s\n", mark, c.Name, c.Detail)
			if c.Fix != "" {
				fmt.Printf("      %s\n", c.Fix)
			}
		}
	}
	return errfmt.Config("not ready to order")
}
