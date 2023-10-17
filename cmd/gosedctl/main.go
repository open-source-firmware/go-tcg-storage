package main

import (
	"github.com/alecthomas/kong"
	plugins "github.com/matfax/go-tcg-storage/pkg/cli"
)

const (
	programName = "gosedctl"
	programDesc = "Go SED control"
)

func main() {
	// Parse kong flags and sub-commands
	ctx := kong.Parse(&cli,
		kong.Name(programName),
		kong.Description(programDesc),
		kong.UsageOnError(),
		kong.Resolvers(plugins.ResolvePassword()),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	// Run the command
	err := ctx.Run(&context{})
	ctx.FatalIfErrorf(err)
}
