package cmdutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
	"golang.org/x/term"
)

// ResolvePassword returns a kong.Resolver that prompts for a password.
// If confirm is true, the user is prompted to enter the password twice for confirmation.
func ResolvePassword(confirm bool) kong.Resolver {
	return kong.ResolverFunc(func(ctx *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		if flag.Tag.Type != "password" || !flag.Required || flag.Value.Set && !flag.Value.Target.IsZero() {
			return nil, nil
		}

		if flag.Target.Kind() != reflect.String {
			return nil, fmt.Errorf(`'password' type must be applied to a string not %s`, flag.Target.Type())
		}

		fmt.Printf("No value has been provided for flag `%s`.\n", flag.ShortSummary())
		if flag.Help != "" {
			fmt.Println("Description: " + flag.Help)
		}

		for {
			fmt.Printf("Enter %s: ", strings.ToTitle(flag.Name))
			bytePassword, err := term.ReadPassword(0)
			fmt.Print("\n")
			if err != nil {
				return "", fmt.Errorf("password could not be read: %v", err)
			}
			pwd := strings.TrimSpace(string(bytePassword))
			if pwd == "" {
				return nil, nil
			}

			if confirm {
				fmt.Printf("Re-enter %s: ", strings.ToTitle(flag.Name))
				bytePassword2, err2 := term.ReadPassword(0)
				fmt.Print("\n\n")
				if err2 != nil {
					return "", fmt.Errorf("password could not be read: %v", err2)
				}
				pwd2 := strings.TrimSpace(string(bytePassword2))
				if pwd != pwd2 {
					fmt.Println("Passwords do not match. Please try again.")
					continue
				}
			}

			return pwd, nil
		}
	})
}
