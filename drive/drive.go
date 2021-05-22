// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"errors"
)

var (
	ErrNotSupported = errors.New("Operation is not supported")
)

type ComID int

type SecurityProtocol int

const (
	ComIDDiscoveryL0 ComID = 1

	SecurityProtocolManagement SecurityProtocol = 1
	SecurityProtocolTPer       SecurityProtocol = 2
)

type driveIntf interface {
	IFRecv(proto SecurityProtocol, comID ComID, data *[]byte) error
	IFSend(proto SecurityProtocol, comID ComID, data []byte) error

	Close() error
}
