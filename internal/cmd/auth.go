package cmd

import (
	"fmt"
	"os"

	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
	"github.com/zalando/go-keyring"
)

type AuthLoginCmd struct {
	Token string `help:"Provide JWT token directly (skip browser login)." env:"ZONASUL_TOKEN"`
}

type AuthStatusCmd struct{}
type AuthLogoutCmd struct{}

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Login via browser (Zona Sul OAuth)."`
	Status AuthStatusCmd `cmd:"" help:"Check auth state and token expiry."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear stored credentials."`
}

func (c *AuthLoginCmd) Run(g *Globals) error {
	var jwt string

	if c.Token != "" {
		jwt = c.Token
	} else if g.CLI.NoInput {
		return errfmt.Usage("use --token or ZONASUL_TOKEN to provide a JWT in non-interactive mode")
	} else {
		fmt.Fprintln(os.Stderr, "Zona Sul uses custom OAuth login.")
		fmt.Fprintln(os.Stderr, "Choose login method:")
		fmt.Fprintln(os.Stderr, "  1) Paste JWT token from browser (DevTools > Application > Cookies > VtexIdclientAutCookie_zonasul)")
		fmt.Fprintln(os.Stderr, "  2) Open browser for OAuth login (experimental)")
		choice := readLine("Choice [1]: ")
		if choice == "2" {
			client := vtex.NewClient(vtex.BaseURL, "")
			var err error
			jwt, err = client.OAuthLogin()
			if err != nil {
				return err
			}
		} else {
			jwt = readLine("Paste JWT token: ")
			if jwt == "" {
				return errfmt.Usage("no token provided")
			}
		}
	}

	if err := keyring.Set(keyringService, keyringUser, jwt); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to store token in keychain", err)
	}

	client := vtex.NewClient(vtex.BaseURL, jwt)
	user, err := client.AuthenticatedUser()
	if err != nil {
		outfmt.Warn("token may be invalid or expired")
	} else {
		outfmt.Success("Logged in as: %s", user)
	}

	outfmt.Hint("Token stored in keychain.")

	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "ok"})
	}
	return nil
}

func (c *AuthStatusCmd) Run(g *Globals) error {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil || token == "" {
		if g.CLI.JSON {
			_ = g.Formatter().Print(map[string]string{"status": "unauthenticated"})
		}
		return errfmt.Auth("not logged in (run: zonasul auth login)")
	}

	client := vtex.NewClient(vtex.BaseURL, token)
	user, err := client.AuthenticatedUser()
	if err != nil {
		if g.CLI.JSON {
			_ = g.Formatter().Print(map[string]string{"status": "expired"})
		}
		return errfmt.Auth("token expired or invalid (run: zonasul auth login)")
	}

	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "ok", "user": user})
	}
	outfmt.Success("Logged in as: %s", user)
	return nil
}

func (c *AuthLogoutCmd) Run(g *Globals) error {
	_ = keyring.Delete(keyringService, keyringUser)
	outfmt.Success("Logged out. Token removed from keychain.")
	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "logged_out"})
	}
	return nil
}
