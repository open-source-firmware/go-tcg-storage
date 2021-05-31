// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"

	"github.com/bluecmd/go-tcg-storage/pkg/drive/sgio"
)

type scsiDrive struct {
	fd uintptr
}

func (d *scsiDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	// TODO: It seems that some drives are picky on that the data is aligned in some fashion, possibly to 512?
	// Should work something out to ensure we pad the request accordingly
	err := sgio.SCSISecurityIn(d.fd, uint8(proto), sps, data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	// TODO: It seems that some drives are picky on that the data is aligned in some fashion, possibly to 512?
	// Should work something out to ensure we pad the request accordingly
	err := sgio.SCSISecurityOut(d.fd, uint8(proto), sps, data)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) Identify() (string, error) {
	id, err := sgio.SCSIInquiry(d.fd)
	if err != nil {
		return "", err
	}
	return "Protocol=SCSI, " + id.String(), nil
}

func (d *scsiDrive) SerialNumber() ([]byte, error) {
	id, err := sgio.SCSIInquiry(d.fd)
	if err != nil {
		return nil, err
	}
	return id.SerialNumber[:], nil
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
