// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style // license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrNotSupported       = errors.New("operation is not supported")
	ErrDeviceNotSupported = errors.New("device is not supported")
)

type SecurityProtocol int

const (
	SecurityProtocolInformation   SecurityProtocol = 0
	SecurityProtocolTCGManagement SecurityProtocol = 1
	SecurityProtocolTCGTPer       SecurityProtocol = 2
)

type Identity struct {
	Protocol     string
	SerialNumber string
	Model        string
	Firmware     string
}

func (i *Identity) String() string {
	return fmt.Sprintf("Protocol=%s, Model=%s, Serial=%s, Firmware=%s",
		i.Protocol, i.Model, i.SerialNumber, i.Firmware)
}

type DriveIntf interface {
	SendReceive
	Identify
	Closer
}

type SendReceive interface {
	IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error
	IFSend(proto SecurityProtocol, sps uint16, data []byte) error
}

type Identify interface {
	Identify() (*Identity, error)
	SerialNumber() ([]byte, error)
}

type Closer interface {
	Close() error
}

// Returns a list of supported security protocols.
func SecurityProtocols(d DriveIntf) ([]SecurityProtocol, error) {
	raw := make([]byte, 2048)
	if err := d.IFRecv(SecurityProtocolInformation, 0, &raw); err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(raw)
	hdr := struct {
		_      [6]byte
		Length uint16
	}{}
	if err := binary.Read(buf, binary.BigEndian, &hdr); err != nil {
		return nil, fmt.Errorf("failed to parse security protocol list header: %v", err)
	}
	i := hdr.Length
	list := make([]uint8, i)
	if err := binary.Read(buf, binary.BigEndian, list); err != nil {
		return nil, fmt.Errorf("failed to read security protocol list: %v", err)
	}
	res := []SecurityProtocol{}
	for _, i := range list {
		res = append(res, SecurityProtocol(i))
	}
	return res, nil
}

// Returns the X.509 security certificate from the drive.
func Certificate(d DriveIntf) ([]*x509.Certificate, error) {
	raw := make([]byte, 4096)
	if err := d.IFRecv(SecurityProtocolInformation, 1, &raw); err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(raw)
	hdr := struct {
		_    uint16
		Size uint16
	}{}
	if err := binary.Read(buf, binary.BigEndian, &hdr); err != nil {
		return nil, fmt.Errorf("failed to parse certificate header: %v", err)
	}
	if hdr.Size == 0 {
		return nil, nil
	}
	crtdata := make([]byte, hdr.Size)
	if n, err := buf.Read(crtdata); n != int(hdr.Size) || err != nil {
		return nil, fmt.Errorf("failed to read certificate: error (%v) or underrun", err)
	}
	return x509.ParseCertificates(crtdata)
}
