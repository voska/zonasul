package command

import "github.com/alecthomas/kong"

// Options keeps the shared command tree while scoping raw tokens to the BFF.
func Options(version string) []kong.Option {
	return []kong.Option{
		kong.Name("zonasul"), kong.Description("Zona Sul supermarket CLI."), kong.UsageOnError(), kong.Vars{"version": version}, kong.DefaultEnvars("ZONASUL"),
		kong.PostBuild(func(parser *kong.Kong) error {
			var walk func(*kong.Node)
			walk = func(node *kong.Node) {
				if node.Path() == "auth login" {
					node.Help = "Log in to the current storefront. Run doctor to check readiness."
					for _, flag := range node.Flags {
						if flag.Name == "token" {
							flag.Envs = []string{"ZONASUL_BFF_TOKEN"}
							flag.Tag.Envs = flag.Envs
							flag.Help = "Use a BFF token with --email for identity verification."
						}
					}
				}
				if node.Path() == "cart add" {
					node.Help = "Add packs using a product:<UUID> handle from search. Legacy numeric SKUs are unsupported."
				}
				if node.Path() == "product" {
					node.Help = "Show a product:<UUID> handle from search. Legacy numeric SKUs are unsupported."
				}
				if node.Path() == "checkout run" {
					for _, flag := range node.Flags {
						switch flag.Name {
						case "confirm", "payment", "window", "cvv":
							flag.Envs = nil
							flag.Tag.Envs = nil
						}
						if flag.Name == "payment" {
							flag.Help = "Saved credit card ID from checkout payments. Pix/cash unsupported."
						}
						if flag.Name == "cvv" {
							flag.Help = "Unsupported on BFF; do not supply card security codes."
						}
					}
				}
				for _, child := range node.Children {
					walk(child)
				}
			}
			walk(parser.Model.Node)
			return nil
		}),
	}
}
