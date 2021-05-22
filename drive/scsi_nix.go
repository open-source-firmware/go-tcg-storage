// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"github.com/bluecmd/go-opal/drive/sgio"
)

func isSCSI(fd FdIntf) bool {
	_, err := sgio.InquirySCSI(fd.Fd())
	return err == nil
}
