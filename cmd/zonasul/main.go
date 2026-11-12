package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/alecthomas/kong"
	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/vtexkit/cli/outfmt"
	"github.com/voska/zonasul"
	"github.com/voska/zonasul/internal/command"
	"golang.org/x/term"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	var args cli.CLI
	ctx := kong.Parse(&args, command.Options(version)...)
	args.NoInput = args.NoInput || !term.IsTerminal(int(os.Stdin.Fd()))
	g := &cli.Globals{CLI: &args, Store: zonasul.Store, Version: version}
	handled, err := (&command.Runner{}).Run(ctx.Command(), g)
	if !handled {
		err = ctx.Run(g)
	}
	if err != nil {
		var e *errfmt.Error
		if !errors.As(err, &e) {
			e = errfmt.New(errfmt.ExitError, err.Error())
		}
		if args.JSON {
			_ = json.NewEncoder(os.Stderr).Encode(e)
		} else {
			outfmt.ErrorMsg("%s", e)
		}
		os.Exit(e.Code)
	}
}
