// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"

	"github.com/matfax/go-tcg-storage/pkg/drive/sgio"
)

type scsiDrive struct {
	fd FdIntf
}

func (d *scsiDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	// TODO: It seems that some drives are picky on that the data is aligned in some fashion, possibly to 512?
	// Should work something out to ensure we pad the request accordingly
	err := sgio.SCSISecurityIn(d.fd.Fd(), uint8(proto), sps, data)
	runtime.KeepAlive(d.fd)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	// TODO: It seems that some drives are picky on that the data is aligned in some fashion, possibly to 512?
	// Should work something out to ensure we pad the request accordingly
	err := sgio.SCSISecurityOut(d.fd.Fd(), uint8(proto), sps, data)
	runtime.KeepAlive(d.fd)
	if err == sgio.ErrIllegalRequest {
		return ErrNotSupported
	}
	return err
}

func (d *scsiDrive) Identify() (*Identity, error) {
	id, err := sgio.SCSIInquiry(d.fd.Fd())
	runtime.KeepAlive(d.fd)
	if err != nil {
		return nil, err
	}

	m := ""
	protocol := ""
	if bytes.Equal(id.VendorIdent, []byte("ATA     ")) {
		// SCSI ATA Translation (SAT)
		protocol = "SATA"
		m = strings.TrimSpace(string(id.ProductIdent))
	} else {
		protocol = id.Protocol.String()
		m = fmt.Sprintf("%s %s",
			strings.TrimSpace(string(id.VendorIdent)),
			strings.TrimSpace(string(id.ProductIdent)))
	}

	return &Identity{
		Protocol:     protocol,
		Model:        m,
		Firmware:     strings.TrimSpace(string(id.ProductRev)),
		SerialNumber: strings.TrimSpace(string(id.SerialNumber)),
	}, nil
}

func (d *scsiDrive) SerialNumber() ([]byte, error) {
	id, err := sgio.SCSIInquiry(d.fd.Fd())
	runtime.KeepAlive(d.fd)
	if err != nil {
		return nil, err
	}
	return id.SerialNumber[:], nil
}

func (d *scsiDrive) Close() error {
	return d.fd.Close()
}

func SCSIDrive(fd FdIntf) *scsiDrive {
	// Save the full object reference to avoid the underlying File-like object
	// to be GC'd
	return &scsiDrive{fd: fd}
}

func isSCSI(fd FdIntf) bool {
	_, err := sgio.SCSIInquiry(fd.Fd())
	return err == nil
}
