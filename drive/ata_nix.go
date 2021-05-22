// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"

	"github.com/u-root/u-root/pkg/mount/scuzz"
)

func isATA(fd FdIntf) bool {
	f := os.NewFile(fd.Fd(), "dummy")
	_, err := scuzz.NewSGDiskFromFile(f)
	return err == nil
}
