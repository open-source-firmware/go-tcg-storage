package kong

import (
	"fmt"
	"os"
	"reflect"

	"github.com/alecthomas/kong"
)

func AccessibleFileMapper() kong.MapperFunc {
	return func(ctx *kong.DecodeContext, target reflect.Value) error {

		if target.Kind() != reflect.String {
			return fmt.Errorf(`"accessiblefile" type must be applied to a string not %s`, target.Type())
		}
		var path string
		err := ctx.Scan.PopValueInto("file", &path)
		if err != nil {
			return err
		}

		if path != "-" {
			path = kong.ExpandPath(path)
			stat, err := os.Stat(path)
			if err != nil {
				if os.IsPermission(err) {
					return fmt.Errorf("permission denied for file %q", path)
				}
				return err
			}
			if stat.IsDir() {
				return fmt.Errorf("%q exists but is a directory", path)
			}
		}
		target.SetString(path)
		return nil
	}
}
