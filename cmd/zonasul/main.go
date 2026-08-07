package main

import (
	"github.com/voska/vtexkit/cli"
	"github.com/voska/zonasul"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	cli.Main(cli.App{
		Store:       zonasul.Store,
		Version:     version,
		Description: "Zona Sul supermarket CLI for humans and AI agents.",
	})
}
