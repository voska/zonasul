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

type CLI struct {
	JSON    bool `help:"Output JSON for agent consumption." env:"ZONASUL_JSON"`
	Plain   bool `help:"Output plain text for piping." env:"ZONASUL_PLAIN"`
	NoInput bool `help:"Disable interactive prompts." env:"ZONASUL_NO_INPUT"`
	Version kong.VersionFlag `help:"Print version and exit."`

	Auth     AuthCmd     `cmd:"" help:"Authentication commands."`
	Search   SearchCmd   `cmd:"" help:"Search products."`
	Cart     CartCmd     `cmd:"" help:"Manage shopping cart."`
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
		return nil, errfmt.Auth("not logged in (run: zonasul auth login)")
	}
	return vtex.NewClient(vtex.BaseURL, token), nil
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

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
