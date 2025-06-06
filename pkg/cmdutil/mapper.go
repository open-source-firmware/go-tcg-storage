// Copyright (C) 2018 Alec Thomas
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// Modifications made by Matthias Fax, 2025.

package cmdutil

import (
	"fmt"
	"os"
	"reflect"

	"github.com/alecthomas/kong"
)

// AccessibleFileMapper is restrictive version of kong.existingFileMapper
// that does not return nil (effectively an empty string) on permission denied errors
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
