// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"fmt"
)

// NewSATTransport returns an ATATransport sending ATA commands through t,
// wrapped in ATA PASS-THROUGH commands for a SCSI/ATA Translation (SAT) layer
// to pass on to the drive.
//
// SATA drives behind a SAT layer normally work with NewSCSIDrive, as the layer
// translates the SCSI security commands itself. This transport is for testing
// NewATADrive on such systems, and for drives whose SAT layer does not
// translate the security commands. Linux libata only passes TRUSTED commands
// through when booted with libata.allow_tpm=1.
func NewSATTransport(t SCSITransport) ATATransport {
	return &satTransport{t: t}
}

const scsiOpATAPassThrough16 = 0x85

// ATA PASS-THROUGH PROTOCOL field values
const (
	satProtocolPIODataIn  = 4
	satProtocolPIODataOut = 5
)

// satDescATAStatusReturn is the sense data descriptor carrying the ATA
// registers after an ATA PASS-THROUGH command.
const satDescATAStatusReturn = 0x09

type satTransport struct {
	t SCSITransport
}

func (s *satTransport) SendCommand(cmd ATACommand, data []byte) (ATAResult, error) {
	cdb, dir, err := satPassThroughCDB(cmd)
	if err != nil {
		return ATAResult{}, err
	}
	res, err := s.t.SendCDB(cdb, dir, data)
	if err != nil {
		return ATAResult{}, err
	}
	return satResult(res)
}

func (s *satTransport) Close() error {
	return s.t.Close()
}

// satPassThroughCDB returns a 16-byte ATA PASS-THROUGH CDB for cmd, and the
// direction of its data transfer. ATACommand does not say whether cmd is a
// 48-bit command, so EXTEND is set if it uses any of the 48-bit fields.
func satPassThroughCDB(cmd ATACommand) ([]byte, SCSIDirection, error) {
	var protocol, tDir uint8
	var dir SCSIDirection
	switch cmd.Protocol {
	case ATAPIODataIn:
		protocol, tDir, dir = satProtocolPIODataIn, 1, SCSIDataIn
	case ATAPIODataOut:
		protocol, tDir, dir = satProtocolPIODataOut, 0, SCSIDataOut
	default:
		return nil, 0, fmt.Errorf("unsupported ATA protocol %d", cmd.Protocol)
	}

	var extend uint8
	if cmd.Features > 0xff || cmd.Count > 0xff || cmd.LBA > 0xffffff {
		extend = 1
	}

	cdb := make([]byte, 16)
	cdb[0] = scsiOpATAPassThrough16
	cdb[1] = protocol<<1 | extend
	// CK_COND clear, T_DIR set for transfers from the device, and T_LENGTH 3
	// to take the transfer length from the data buffer, as each ATA command
	// encodes it in different fields
	cdb[2] = tDir<<3 | 0x3
	cdb[3] = uint8(cmd.Features >> 8)
	cdb[4] = uint8(cmd.Features)
	cdb[5] = uint8(cmd.Count >> 8)
	cdb[6] = uint8(cmd.Count)
	cdb[7] = uint8(cmd.LBA >> 24)
	cdb[8] = uint8(cmd.LBA)
	cdb[9] = uint8(cmd.LBA >> 32)
	cdb[10] = uint8(cmd.LBA >> 8)
	cdb[11] = uint8(cmd.LBA >> 40)
	cdb[12] = uint8(cmd.LBA >> 16)
	cdb[13] = cmd.Device
	cdb[14] = cmd.Command
	return cdb, dir, nil
}

// satResult returns the outcome of an ATA PASS-THROUGH command. A successful
// command returns no registers, so its result is zero. An unsuccessful command
// returns them in the sense data. If they are missing, the device did not
// execute the command, e.g. because the SAT layer rejected it.
func satResult(res SCSIResult) (ATAResult, error) {
	if res.Status == scsiStatusGood {
		return ATAResult{}, nil
	}
	if r, ok := satSenseRegisters(res.Sense); ok {
		return r, nil
	}
	// Not wrapped: the device did not execute the command, so this says nothing
	// about whether it supports it
	return ATAResult{}, fmt.Errorf("ATA PASS-THROUGH failed: %v", scsiResultError(res))
}

// satSenseRegisters returns the ATA Status and Error fields from sense data.
// Descriptor format sense data carries them in an ATA Status Return
// descriptor, fixed format sense data in its information field, which is only
// valid with the VALID bit set.
func satSenseRegisters(sense []byte) (ATAResult, bool) {
	if len(sense) < 1 {
		return ATAResult{}, false
	}
	switch sense[0] {
	case 0xf0: // fixed, VALID
		if len(sense) < 5 {
			return ATAResult{}, false
		}
		return ATAResult{Error: sense[3], Status: sense[4]}, true
	case 0x72: // descriptor
		if len(sense) < 8 {
			return ATAResult{}, false
		}
		descs := sense[8:min(8+int(sense[7]), len(sense))]
		for len(descs) >= 2 {
			l := int(descs[1]) + 2 // including the 2 byte header
			if l > len(descs) {
				break
			}
			if descs[0] == satDescATAStatusReturn && l >= 14 {
				return ATAResult{Error: descs[3], Status: descs[13]}, true
			}
			descs = descs[l:]
		}
	}
	return ATAResult{}, false
}
