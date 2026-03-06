package cmd

type SchemaCmd struct{}

func (c *SchemaCmd) Run(g *Globals) error {
	return g.Formatter().Print(fullSchema(g.Version))
}

func fullSchema(version string) map[string]any {
	return map[string]any{
		"name":    "zonasul",
		"version": version,
		"commands": []map[string]any{
			{
				"name": "auth",
				"help": "Authentication commands",
				"subcommands": []map[string]any{
					{"name": "login", "help": "Login via browser or JWT token", "flags": []string{"--token"}},
					{"name": "status", "help": "Check auth state and token expiry"},
					{"name": "logout", "help": "Clear stored credentials"},
				},
			},
			{
				"name":  "search",
				"help":  "Search products",
				"args":  []string{"query"},
				"flags": []string{"--limit"},
			},
			{
				"name": "cart",
				"help": "Manage shopping cart",
				"subcommands": []map[string]any{
					{"name": "show", "help": "Show current cart contents (default)"},
					{"name": "add", "help": "Add item to cart", "args": []string{"sku"}, "flags": []string{"--qty"}},
					{"name": "remove", "help": "Remove item from cart", "args": []string{"index"}},
					{"name": "clear", "help": "Clear all items from cart"},
				},
			},
			{
				"name": "delivery",
				"help": "Delivery options",
				"subcommands": []map[string]any{
					{"name": "windows", "help": "List available delivery windows"},
				},
			},
			{
				"name": "checkout",
				"help": "Place an order",
				"flags": []map[string]any{
					{"name": "--window", "type": "int", "help": "Delivery window index"},
					{"name": "--payment", "type": "enum", "values": []string{"pix", "credit", "cash", "vr", "alelo", "ticket"}, "default": "pix", "help": "Payment method"},
					{"name": "--cvv", "type": "string", "help": "Credit card CVV (required for --payment credit)", "env": "ZONASUL_CVV"},
					{"name": "--confirm", "type": "bool", "help": "Actually place the order. Required safety gate."},
				},
			},
			{
				"name": "orders",
				"help": "List recent orders",
			},
			{
				"name": "agent",
				"help": "Agent introspection commands",
				"subcommands": []map[string]any{
					{"name": "exit-codes", "help": "Print exit code reference table"},
				},
			},
			{
				"name": "schema",
				"help": "Dump CLI schema as JSON for agent introspection",
			},
		},
		"global_flags": []map[string]any{
			{"name": "--json", "help": "Output JSON for agent consumption", "env": "ZONASUL_JSON"},
			{"name": "--plain", "help": "Output plain text for piping", "env": "ZONASUL_PLAIN"},
			{"name": "--no-input", "help": "Disable interactive prompts", "env": "ZONASUL_NO_INPUT"},
		},
	}
}
