package kong

import (
	"fmt"
	"github.com/alecthomas/kong"
	"golang.org/x/term"
	"strings"
)

func ResolvePassword() kong.Resolver {
	return kong.ResolverFunc(func(ctx *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		if flag.Tag.Type != "password" || flag.Tag.TypeName != "string" || !flag.Required {
			return nil, nil
		}

		pwd := ctx.FlagValue(flag).(string)

		fmt.Printf("No value has been provided for flag `%s`.\n", flag.ShortSummary())
		if flag.Help != "" {
			fmt.Println("Description: " + flag.Help)
		}
		fmt.Printf("Enter %s: ", strings.ToTitle(flag.Name))

		bytePassword, err := term.ReadPassword(0)

		fmt.Print("\n\n")
		if err != nil {
			return "", fmt.Errorf("password could not be read: %v", err)
		}

		pwd = string(bytePassword)
		if pwd == "" {
			return nil, nil
		}

		return pwd, nil
	})
}
