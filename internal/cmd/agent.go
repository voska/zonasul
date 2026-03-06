package cmd

import (
	"github.com/voska/zonasul/internal/errfmt"
)

type AgentExitCodesCmd struct{}

type AgentCmd struct {
	ExitCodes AgentExitCodesCmd `cmd:"exit-codes" help:"Print exit code reference table."`
}

func (c *AgentExitCodesCmd) Run(g *Globals) error {
	return g.Formatter().Print(errfmt.ExitCodeTable())
}
