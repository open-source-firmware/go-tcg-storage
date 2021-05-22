// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"

	"github.com/bluecmd/go-opal/drive/sgio"
)

type ataDrive struct {
	fd uintptr
}

func (d *ataDrive) IFRecv(proto SecurityProtocol, comID ComID, data *[]byte) error {
	err := sgio.ATATrustedReceive(d.fd, uint8(proto), uint16(comID), data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *ataDrive) IFSend(proto SecurityProtocol, comID ComID, data []byte) error {
	err := sgio.ATATrustedSend(d.fd, uint8(proto), uint16(comID), data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *ataDrive) Close() error {
	return os.NewFile(d.fd, "").Close()
}

func isATA(fd FdIntf) bool {
	_, err := sgio.ATAIdentify(fd.Fd())
	return err == nil
}

func ATADrive(fd FdIntf) *ataDrive {
	return &ataDrive{fd: fd.Fd()}
}
