package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/alecthomas/kong"
	"github.com/voska/zonasul/internal/cmd"
	"github.com/voska/zonasul/internal/errfmt"
	"github.com/voska/zonasul/internal/outfmt"
	"golang.org/x/term"
)

var version = "dev"

var cli cmd.CLI

func main() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		cli.NoInput = true
	}

	ctx := kong.Parse(&cli,
		kong.Name("zonasul"),
		kong.Description("Zona Sul supermarket CLI for AI agents."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
	)

	globals := cmd.NewGlobals(&cli, version)

	err := ctx.Run(globals)
	if err != nil {
		handleError(err)
	}
}

func handleError(err error) {
	var e *errfmt.Error
	if errors.As(err, &e) {
		if cli.JSON {
			_ = json.NewEncoder(os.Stderr).Encode(e)
		} else {
			outfmt.ErrorMsg("%s", e.Error())
		}
		os.Exit(e.Code)
	}
	if cli.JSON {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{
			"code":    errfmt.ExitError,
			"message": err.Error(),
		})
	} else {
		outfmt.ErrorMsg("%s", err.Error())
	}
	os.Exit(errfmt.ExitError)
}
