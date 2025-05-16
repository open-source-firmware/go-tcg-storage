package cmdutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
	"golang.org/x/term"
)

func ResolvePassword() kong.Resolver {
	return kong.ResolverFunc(func(ctx *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		if flag.Tag.Type != "password" || !flag.Required {
			return nil, nil
		}

		if flag.Target.Kind() != reflect.String {
			return nil, fmt.Errorf(`'password' type must be applied to a string not %s`, flag.Target.Type())
		}

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

		pwd := string(bytePassword)
		if pwd == "" {
			return nil, nil
		}

		return pwd, nil
	})
}
