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

type SecurityProtocol int

const (
	SecurityProtocolInformation   SecurityProtocol = 0
	SecurityProtocolTCGManagement SecurityProtocol = 1
	SecurityProtocolTCGTPer       SecurityProtocol = 2
)

type driveIntf interface {
	IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error
	IFSend(proto SecurityProtocol, sps uint16, data []byte) error

	Close() error
}
