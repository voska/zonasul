package command

import (
	"fmt"

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
		addresses, addressErr := r.addresses(session.Token)
		defaults := 0
		for _, a := range addresses {
			if a.Default {
				defaults++
			}
		}
		cards, cardErr := r.client().Cards(session.Token)
		for _, item := range []struct {
			name    string
			count   int
			err     error
			missing string
		}{
			{"delivery address", defaults, addressErr, "save one default delivery address on www.zonasul.com.br"},
			{"saved payment methods", len(cards), cardErr, "save a credit card on www.zonasul.com.br"},
		} {
			result := check{Name: item.name, OK: item.err == nil && item.count > 0, Detail: fmt.Sprintf("%d on the BFF account", item.count)}
			if item.name == "delivery address" {
				result.OK = item.err == nil && item.count == 1
			}
			if item.err != nil {
				result.Detail = item.err.Error()
			} else if item.count == 0 {
				result.Fix = item.missing
			}
			checks = append(checks, result)
		}
	}
	checks = append(checks, check{Name: "CLI checkout", OK: true, Detail: "BFF cart, windows and saved-card checkout available; settlement requires issuer verification"})
	ready := true
	for _, c := range checks {
		ready = ready && c.OK
	}
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		if err := g.Formatter().Print(map[string]any{"store": "zonasul", "backend": "bff", "ready": ready, "checks": checks}); err != nil {
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
	if !ready {
		return errfmt.Config("not ready to order")
	}
	return nil
}
