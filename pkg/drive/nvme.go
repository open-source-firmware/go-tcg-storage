// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"fmt"
	"strings"
)

// NVMeAdminCommand is an NVMe admin command, as far as defined by the NVMe
// specification. The transport provides the command identifier and the data
// pointers.
type NVMeAdminCommand struct {
	Opcode uint8
	NSID   uint32
	CDW10  uint32
	CDW11  uint32
	CDW12  uint32
	CDW13  uint32
	CDW14  uint32
	CDW15  uint32
}

// NVMeStatus is the Status Field of a completion queue entry, without the
// Phase Tag: the status code is in bits 7:0, and the status code type in bits
// 10:8. A non-zero NVMeStatus is an error.
type NVMeStatus uint16

func (s NVMeStatus) code() uint8 {
	return uint8(s)
}

func (s NVMeStatus) codeType() uint8 {
	return uint8(s>>8) & 0x7
}

func (s NVMeStatus) Error() string {
	return fmt.Sprintf("NVMe command failed: status code type %#x, status code %#02x", s.codeType(), s.code())
}

// invalidCommand reports whether the controller rejected the command as
// invalid, i.e. with the generic Invalid Command Opcode or Invalid Field in
// Command status, which is how unsupported commands are reported.
func (s NVMeStatus) invalidCommand() bool {
	return s.codeType() == 0 && (s.code() == 0x01 || s.code() == 0x02)
}

// NVMeTransport sends NVMe admin commands to a controller.
type NVMeTransport interface {
	// AdminCommand executes cmd, transferring data in the direction implied
	// by its opcode. The error is for failures to execute the command only, a
	// command completed by the controller reports its outcome in the status,
	// even if unsuccessful.
	AdminCommand(cmd NVMeAdminCommand, data []byte) (NVMeStatus, error)
	Close() error
}

// NewNVMeDrive returns a drive sending NVMe admin commands through t.
func NewNVMeDrive(t NVMeTransport) DriveIntf {
	return &nvmeDrive{t: t}
}

// Admin command opcodes
const (
	nvmeOpIdentify        = 0x06
	nvmeOpSecuritySend    = 0x81
	nvmeOpSecurityReceive = 0x82
)

// nvmeIdentifyLength is the length of Identify data structures.
const nvmeIdentifyLength = 4096

// Identify Controller or Namespace Structure (CNS) values
const nvmeCNSController = 0x01

type nvmeDrive struct {
	t    NVMeTransport
	ctrl []byte // Identify Controller data, cached on first use
}

func (d *nvmeDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	return d.submit(NVMeAdminCommand{
		Opcode: nvmeOpSecurityReceive,
		CDW10:  nvmeSecurityCDW10(uint8(proto), sps),
		CDW11:  uint32(len(*data)), // allocation length
	}, *data)
}

func (d *nvmeDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	return d.submit(NVMeAdminCommand{
		Opcode: nvmeOpSecuritySend,
		CDW10:  nvmeSecurityCDW10(uint8(proto), sps),
		CDW11:  uint32(len(data)), // transfer length
	}, data)
}

// nvmeSecurityCDW10 encodes the security protocol (SECP) in bits 31:24, and
// the SP specific field (SPSP) in bits 23:8.
func nvmeSecurityCDW10(proto uint8, sps uint16) uint32 {
	return uint32(proto)<<24 | uint32(sps)<<8
}

func (d *nvmeDrive) Identify() (*Identity, error) {
	ctrl, err := d.identifyController()
	if err != nil {
		return nil, err
	}
	return &Identity{
		Protocol:     "NVMe",
		Model:        strings.TrimSpace(string(ctrl[24:64])),
		SerialNumber: strings.TrimSpace(string(ctrl[4:24])),
		Firmware:     strings.TrimSpace(string(ctrl[64:72])),
	}, nil
}

func (d *nvmeDrive) SerialNumber() ([]byte, error) {
	ctrl, err := d.identifyController()
	if err != nil {
		return nil, err
	}
	return bytes.Clone(ctrl[4:24]), nil
}

func (d *nvmeDrive) Close() error {
	return d.t.Close()
}

func (d *nvmeDrive) submit(cmd NVMeAdminCommand, data []byte) error {
	if len(data) == 0 {
		return ErrEmptyBuffer
	}
	status, err := d.t.AdminCommand(cmd, data)
	if err != nil {
		return err
	}
	if status != 0 {
		if status.invalidCommand() {
			return ErrNotSupported
		}
		return status
	}
	return nil
}

// identifyController returns the Identify Controller data structure, with the
// serial number at bytes 4-23, the model number at 24-63 and the firmware
// revision at 64-71.
func (d *nvmeDrive) identifyController() ([]byte, error) {
	if d.ctrl != nil {
		return d.ctrl, nil
	}

	buf := make([]byte, nvmeIdentifyLength)
	cmd := NVMeAdminCommand{
		Opcode: nvmeOpIdentify,
		NSID:   0, // the controller, not a namespace
		CDW10:  nvmeCNSController,
	}
	if err := d.submit(cmd, buf); err != nil {
		return nil, err
	}

	d.ctrl = buf
	return buf, nil
}
