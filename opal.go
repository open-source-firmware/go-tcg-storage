// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package opal

import (
	"fmt"

	"github.com/bluecmd/go-opal/drive"
)

type DriveIntf interface {
	IFRecv(cmd drive.Command, proto drive.Protocol, comID drive.ComID, data []byte) error
	IFSend(cmd drive.Command, proto drive.Protocol, comID drive.ComID, data []byte) error
}

type opalSession struct {

}

func Open(drive DriveIntf) (*opalSession, error) {
	return nil, fmt.Errorf("Not implemented")
}
