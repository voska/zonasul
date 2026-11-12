package command

import "github.com/alecthomas/kong"

// Options keeps the shared command tree while scoping raw tokens to the BFF.
func Options(version string) []kong.Option {
	return []kong.Option{
		kong.Name("zonasul"), kong.Description("Zona Sul supermarket CLI. Ordering is disabled during the storefront migration."), kong.UsageOnError(), kong.Vars{"version": version}, kong.DefaultEnvars("ZONASUL"),
		kong.PostBuild(func(parser *kong.Kong) error {
			var walk func(*kong.Node)
			walk = func(node *kong.Node) {
				if node.Path() == "auth login" {
					node.Help = "Log in to the current storefront; ordering is disabled in this build."
					for _, flag := range node.Flags {
						if flag.Name == "token" {
							flag.Envs = []string{"ZONASUL_BFF_TOKEN"}
							flag.Tag.Envs = flag.Envs
							flag.Help = "Use a BFF token with --email for identity verification."
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
