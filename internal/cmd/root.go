package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/voska/zonasul/internal/config"
	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"github.com/voska/zonasul/internal/vtex"
	"github.com/zalando/go-keyring"
)

const keyringService = "zonasul-cli"
const keyringUser = "vtex-jwt"
const keyringPassword = "login-password"

type CLI struct {
	JSON    bool             `help:"Output JSON for agent consumption." env:"ZONASUL_JSON"`
	Plain   bool             `help:"Output plain text for piping." env:"ZONASUL_PLAIN"`
	NoInput bool             `help:"Disable interactive prompts." env:"ZONASUL_NO_INPUT"`
	Version kong.VersionFlag `help:"Print version and exit."`

	Auth     AuthCmd     `cmd:"" help:"Authentication commands."`
	Search   SearchCmd   `cmd:"" help:"Search products."`
	Cart     CartCmd     `cmd:"" help:"Manage shopping cart."`
	List     ListCmd     `cmd:"" help:"Manage named SKU lists."`
	Fav      FavCmd      `cmd:"" help:"Manage favorites (shorthand for list favorites)."`
	Delivery DeliveryCmd `cmd:"" help:"Delivery options."`
	Checkout CheckoutCmd `cmd:"" help:"Place an order."`
	Orders   OrdersCmd   `cmd:"" help:"List recent orders."`
	Agent    AgentCmd    `cmd:"" help:"Agent introspection commands."`
	Schema   SchemaCmd   `cmd:"" help:"Dump CLI schema as JSON for agent introspection."`
}

type Globals struct {
	CLI     *CLI
	Version string
}

func NewGlobals(cli *CLI, version string) *Globals {
	return &Globals{CLI: cli, Version: version}
}

func (g *Globals) Formatter() *outfmt.Formatter {
	return outfmt.FromGlobals(g.CLI.JSON, g.CLI.Plain)
}

func (g *Globals) Client() *vtex.Client {
	token, _ := keyring.Get(keyringService, keyringUser)
	return vtex.NewClient(vtex.BaseURL, token)
}

func (g *Globals) AuthedClient() (*vtex.Client, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil || token == "" {
		// No token — try credential login from stored credentials
		return g.credentialRefresh()
	}
	client := vtex.NewClient(vtex.BaseURL, token)

	// Try to refresh the token via VTEX session API
	newToken, refreshErr := client.RefreshToken()
	if refreshErr == nil && newToken != "" && newToken != token {
		_ = keyring.Set(keyringService, keyringUser, newToken)
		outfmt.Hint("Token refreshed.")
		return client, nil
	}

	// Verify the token is still valid
	if _, authErr := client.AuthenticatedUser(); authErr != nil {
		// Token expired — try credential login
		refreshed, credErr := g.credentialRefresh()
		if credErr != nil {
			return nil, errfmt.Auth("token expired (run: zonasul auth login)")
		}
		return refreshed, nil
	}

	return client, nil
}

func (g *Globals) credentialRefresh() (*vtex.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil || creds.Email == "" {
		return nil, errfmt.Auth("not logged in (run: zonasul auth login)")
	}
	password, err := keyring.Get(keyringService, keyringPassword)
	if err != nil || password == "" {
		return nil, errfmt.Auth("not logged in (run: zonasul auth login)")
	}

	outfmt.Hint("Token expired, re-authenticating...")
	client := vtex.NewClient(vtex.BaseURL, "")
	jwt, err := client.CredentialLogin(creds.Email, password)
	if err != nil {
		return nil, errfmt.Wrap(errfmt.ExitAuth, "auto-refresh failed (run: zonasul auth login)", err)
	}

	_ = keyring.Set(keyringService, keyringUser, jwt)
	outfmt.Hint("Token refreshed.")
	return vtex.NewClient(vtex.BaseURL, jwt), nil
}

func (g *Globals) LoadConfig() (*config.Config, error) {
	return config.Load(config.DefaultPath())
}

func (g *Globals) SaveConfig(cfg *config.Config) error {
	return config.Save(config.DefaultPath(), cfg)
}

func (g *Globals) RequireAuth() (*vtex.Client, error) {
	return g.AuthedClient()
}

// SessionOrderFormID fetches the orderFormId from the VTEX session,
// falling back to the config file if the session doesn't have one.
// Persists the ID to config so it survives across CLI invocations.
func (g *Globals) SessionOrderFormID(client *vtex.Client) string {
	sess, err := client.GetSession()
	if err == nil && sess.OrderFormID != "" {
		g.PersistOrderFormID(sess.OrderFormID)
		return sess.OrderFormID
	}
	cfg, err := g.LoadConfig()
	if err == nil && cfg.OrderFormID != "" {
		return cfg.OrderFormID
	}
	return ""
}

func (g *Globals) PersistOrderFormID(id string) {
	if id == "" {
		return
	}
	cfg, _ := g.LoadConfig()
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.OrderFormID != id {
		cfg.OrderFormID = id
		_ = g.SaveConfig(cfg)
	}
}

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
