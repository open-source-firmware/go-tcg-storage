// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"

	"github.com/bluecmd/go-opal/drive/sgio"
)

type scsiDrive struct {
	fd uintptr
}

func (d *scsiDrive) IFRecv(proto SecurityProtocol, comID ComID, data *[]byte) error {
	err := sgio.SCSISecurityIn(d.fd, uint8(proto), uint16(comID), data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) IFSend(proto SecurityProtocol, comID ComID, data []byte) error {
	err := sgio.SCSISecurityOut(d.fd, uint8(proto), uint16(comID), data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) Close() error {
	return os.NewFile(d.fd, "").Close()
}

func SCSIDrive(fd FdIntf) *scsiDrive {
	return &scsiDrive{fd: fd.Fd()}
}

func isSCSI(fd FdIntf) bool {
	_, err := sgio.SCSIInquiry(fd.Fd())
	return err == nil
}
