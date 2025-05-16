package main

import (
	"github.com/alecthomas/kong"
	"github.com/open-source-firmware/go-tcg-storage/pkg/cmdutil"
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
		kong.Resolvers(cmdutil.ResolvePassword()),
		kong.NamedMapper("accessiblefile", cmdutil.AccessibleFileMapper()),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	// Run the command
	err := ctx.Run(&context{})
	ctx.FatalIfErrorf(err)
}
