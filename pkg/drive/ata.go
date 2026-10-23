// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"fmt"
	"strings"
)

// ATAProtocol is the protocol of an ATA command, which determines how its
// data is transferred.
type ATAProtocol int

const (
	ATAPIODataIn  ATAProtocol = iota // PIO from the device
	ATAPIODataOut                    // PIO to the device
)

// ATACommand is an ATA command, given as the fields of the register set
// defined by the ATA Command Set (ACS). The transport maps them to its own
// encoding, e.g. a Register Host to Device FIS, or an EFI_ATA_COMMAND_BLOCK.
type ATACommand struct {
	Protocol ATAProtocol
	Command  uint8
	Features uint16
	Count    uint16
	LBA      uint64 // bits 47:0
	Device   uint8
}

// ATAResult is the outcome of an ATA command executed by a device: the
// Status and Error fields of the register set after its completion.
type ATAResult struct {
	Status uint8
	Error  uint8
}

// ATATransport sends ATA commands to a device.
type ATATransport interface {
	// SendCommand executes cmd, transferring data as implied by its protocol.
	// The error is for failures to execute the command only, a command
	// completed by the device reports its outcome in the result, even if
	// unsuccessful.
	SendCommand(cmd ATACommand, data []byte) (ATAResult, error)
	Close() error
}

// NewATADrive returns a drive sending ATA commands through t.
func NewATADrive(t ATATransport) DriveIntf {
	return &ataDrive{t: t}
}

// Command codes
const (
	ataCmdTrustedReceive = 0x5c
	ataCmdTrustedSend    = 0x5e
	ataCmdIdentifyDevice = 0xec
)

// ataIdentifyLength is the length of IDENTIFY DEVICE data.
const ataIdentifyLength = 512

// Status and Error field bits
const (
	ataStatusErr = 1 << 0
	ataStatusDF  = 1 << 5 // device fault
	ataErrorAbrt = 1 << 2 // command aborted
)

type ataDrive struct {
	t     ATATransport
	ident []byte // IDENTIFY DEVICE data, cached on first use
}

func (d *ataDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	cmd, err := ataTrustedCommand(ataCmdTrustedReceive, ATAPIODataIn, uint8(proto), sps, len(*data))
	if err != nil {
		return err
	}
	return d.send(cmd, *data)
}

func (d *ataDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	cmd, err := ataTrustedCommand(ataCmdTrustedSend, ATAPIODataOut, uint8(proto), sps, len(data))
	if err != nil {
		return err
	}
	return d.send(cmd, data)
}

func (d *ataDrive) Identify() (*Identity, error) {
	ident, err := d.identifyDevice()
	if err != nil {
		return nil, err
	}
	return &Identity{
		Protocol:     "ATA",
		Model:        strings.TrimSpace(string(ataString(ident[54:94]))),
		SerialNumber: strings.TrimSpace(string(ataString(ident[20:40]))),
		Firmware:     strings.TrimSpace(string(ataString(ident[46:54]))),
	}, nil
}

// SerialNumber returns the 20 byte serial number field with its characters in
// order, which is how Linux libata reports it in the unit serial number VPD
// page.
func (d *ataDrive) SerialNumber() ([]byte, error) {
	ident, err := d.identifyDevice()
	if err != nil {
		return nil, err
	}
	return ataString(ident[20:40]), nil
}

func (d *ataDrive) Close() error {
	return d.t.Close()
}

func (d *ataDrive) send(cmd ATACommand, data []byte) error {
	if len(data) == 0 {
		return ErrEmptyBuffer
	}
	res, err := d.t.SendCommand(cmd, data)
	if err != nil {
		return err
	}
	return ataResultError(res)
}

// identifyDevice returns the IDENTIFY DEVICE data, with the serial number at
// bytes 20-39, the firmware revision at 46-53 and the model number at 54-93.
func (d *ataDrive) identifyDevice() ([]byte, error) {
	if d.ident != nil {
		return d.ident, nil
	}

	buf := make([]byte, ataIdentifyLength)
	cmd := ATACommand{Protocol: ATAPIODataIn, Command: ataCmdIdentifyDevice}
	if err := d.send(cmd, buf); err != nil {
		return nil, err
	}

	d.ident = buf
	return buf, nil
}

// ataResultError returns the error described by an ATA command result,
// ErrNotSupported if the device aborted the command.
func ataResultError(res ATAResult) error {
	if res.Status&(ataStatusErr|ataStatusDF) == 0 {
		return nil
	}
	if res.Status&ataStatusErr != 0 && res.Error&ataErrorAbrt != 0 {
		return ErrNotSupported
	}
	return fmt.Errorf("ATA command failed: status %#02x, error %#02x", res.Status, res.Error)
}

// ataTrustedCommand returns a TRUSTED RECEIVE or SEND command, transferring
// length bytes, which must be a multiple of 512.
func ataTrustedCommand(cmd uint8, protocol ATAProtocol, proto uint8, sps uint16, length int) (ATACommand, error) {
	if length&0x1ff > 0 {
		return ATACommand{}, fmt.Errorf("ATA TRUSTED commands only support 512-byte aligned transfers")
	}
	n := length / 512
	if n > 0xffff {
		return ATACommand{}, fmt.Errorf("ATA TRUSTED transfer of %d bytes is too large", length)
	}
	return ATACommand{
		Protocol: protocol,
		Command:  cmd,
		Features: uint16(proto),
		// The transfer length in 512 byte blocks is split between COUNT (7:0)
		// and LBA (7:0), followed by the SP specific field in LBA (23:8)
		Count: uint16(n & 0xff),
		LBA:   uint64(n>>8) | uint64(sps)<<8,
	}, nil
}

// ataString returns an ATA string with its characters in order. ATA strings
// are stored in 16-bit little-endian words, with the first character of each
// pair in the high byte.
func ataString(b []byte) []byte {
	s := bytes.Clone(b)
	for i := 0; i+1 < len(s); i += 2 {
		s[i], s[i+1] = s[i+1], s[i]
	}
	return s
}
