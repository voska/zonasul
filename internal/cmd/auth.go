package cmd

import (
	"fmt"
	"os"

	"github.com/voska/zonasul/internal/config"
	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
	"github.com/zalando/go-keyring"
)

type AuthLoginCmd struct {
	Token    string `help:"Provide JWT token directly (skip browser login)." env:"ZONASUL_TOKEN"`
	Email    string `help:"Email for credential login." env:"ZONASUL_EMAIL"`
	Password string `help:"Password for credential login." env:"ZONASUL_PASSWORD"`
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
	var email, password string

	if c.Token != "" {
		jwt = c.Token
	} else if c.Email != "" && c.Password != "" {
		email, password = c.Email, c.Password
		outfmt.Hint("Logging in with email/password...")
		client := vtex.NewClient(vtex.BaseURL, "")
		var err error
		jwt, err = client.CredentialLogin(email, password)
		if err != nil {
			return errfmt.Wrap(errfmt.ExitAuth, "credential login failed", err)
		}
	} else if g.CLI.NoInput {
		return errfmt.Usage("use --token, or --email/--password, or env vars ZONASUL_TOKEN / ZONASUL_EMAIL+ZONASUL_PASSWORD")
	} else {
		fmt.Fprintln(os.Stderr, "Zona Sul uses custom OAuth login.")
		fmt.Fprintln(os.Stderr, "Choose login method:")
		fmt.Fprintln(os.Stderr, "  1) Email and password")
		fmt.Fprintln(os.Stderr, "  2) Paste JWT token from browser")
		fmt.Fprintln(os.Stderr, "  3) Open browser for OAuth login (experimental)")
		choice := readLine("Choice [1]: ")
		switch choice {
		case "2":
			jwt = readLine("Paste JWT token: ")
			if jwt == "" {
				return errfmt.Usage("no token provided")
			}
		case "3":
			client := vtex.NewClient(vtex.BaseURL, "")
			var err error
			jwt, err = client.OAuthLogin()
			if err != nil {
				return err
			}
		default:
			email = readLine("Email: ")
			if email == "" {
				return errfmt.Usage("no email provided")
			}
			password = readLine("Password: ")
			if password == "" {
				return errfmt.Usage("no password provided")
			}
			outfmt.Hint("Logging in...")
			client := vtex.NewClient(vtex.BaseURL, "")
			var err error
			jwt, err = client.CredentialLogin(email, password)
			if err != nil {
				return errfmt.Wrap(errfmt.ExitAuth, "credential login failed", err)
			}
		}
	}

	if err := keyring.Set(keyringService, keyringUser, jwt); err != nil {
		return errfmt.Wrap(errfmt.ExitConfig, "failed to store token in keychain", err)
	}

	// Persist credentials for auto-refresh (email in config, password in keyring)
	if email != "" && password != "" {
		_ = config.SaveCredentials(&config.Credentials{Email: email})
		_ = keyring.Set(keyringService, keyringPassword, password)
		outfmt.Hint("Credentials saved for auto-refresh.")
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
	_ = keyring.Delete(keyringService, keyringPassword)
	_ = config.SaveCredentials(&config.Credentials{})
	outfmt.Success("Logged out. Token and credentials removed.")
	if g.CLI.JSON {
		return g.Formatter().Print(map[string]string{"status": "logged_out"})
	}
	return nil
}
