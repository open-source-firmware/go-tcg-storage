// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package opal

import (
	"fmt"
)

type Drive interface {
	SecurityCommand() error
}

type opalSession struct {

}

func Open(drive Drive) (*opalSession, error) {
	return nil, fmt.Errorf("Not implemented")
}
