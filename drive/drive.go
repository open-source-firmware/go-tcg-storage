// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

type Command int

type ComID int

type Protocol int

type driveIntf interface {
	IFRecv(cmd Command, proto Protocol, comID ComID, data []byte) error
	IFSend(cmd Command, proto Protocol, comID ComID, data []byte) error
}
