// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Partially copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Partially copyright 2021 Christian Svensson. All rights reserved.
//
// These portions are licensed under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

// SCSIDirection is the direction of the data transfer of a SCSI command.
type SCSIDirection int

const (
	SCSIDataIn  SCSIDirection = iota // from the device
	SCSIDataOut                      // to the device
)

// SCSIResult is the outcome of a SCSI command executed by a device.
type SCSIResult struct {
	Status uint8  // SCSI status
	Sense  []byte // sense data, if any
}

// SCSITransport sends SCSI commands to a device.
type SCSITransport interface {
	// SendCDB executes cdb, transferring data in the given direction. The
	// error is for failures to execute the command only, a command completed
	// by the device reports its outcome in the result, even if unsuccessful.
	SendCDB(cdb []byte, dir SCSIDirection, data []byte) (SCSIResult, error)
	Close() error
}

// NewSCSIDrive returns a drive sending SCSI commands through t. This includes
// SATA drives behind a SCSI/ATA Translation (SAT) layer, like Linux libata.
func NewSCSIDrive(t SCSITransport) DriveIntf {
	return &scsiDrive{t: t}
}

// Operation codes
const (
	scsiOpInquiry             = 0x12
	scsiOpSecurityProtocolIn  = 0xa2
	scsiOpSecurityProtocolOut = 0xb5
)

// VPD pages
const (
	scsiVPDSupportedPages       = 0x00
	scsiVPDUnitSerialNumber     = 0x80
	scsiVPDDeviceIdentification = 0x83
)

// Allocation lengths
const (
	scsiInquiryLength                 = 0x24 // minimum length of standard INQUIRY data according to SPC-3 (and newer)
	scsiVPDLength                     = 0xff // max page size - should be enough for most VPDs
	scsiDeviceIdentificationVPDLength = 2048
)

const (
	scsiStatusGood             = 0x00
	scsiSenseKeyIllegalRequest = 0x5
)

type scsiDrive struct {
	t   SCSITransport
	inq *scsiInquiry // cached on first use
}

// scsiInquiry is what the INQUIRY data and VPD pages report about a device.
type scsiInquiry struct {
	vendor   []byte
	product  []byte
	revision []byte
	serial   []byte
	protocol scsiProtocol
}

func (d *scsiDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	cdb, err := scsiSecurityProtocolCDB(scsiOpSecurityProtocolIn, uint8(proto), sps, len(*data))
	if err != nil {
		return err
	}
	return d.send(cdb, SCSIDataIn, *data)
}

func (d *scsiDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	cdb, err := scsiSecurityProtocolCDB(scsiOpSecurityProtocolOut, uint8(proto), sps, len(data))
	if err != nil {
		return err
	}
	return d.send(cdb, SCSIDataOut, data)
}

func (d *scsiDrive) Identify() (*Identity, error) {
	inq, err := d.inquire()
	if err != nil {
		return nil, err
	}

	m := ""
	protocol := ""
	if bytes.Equal(inq.vendor, []byte("ATA     ")) {
		// SCSI ATA Translation (SAT)
		protocol = "SATA"
		m = strings.TrimSpace(string(inq.product))
	} else {
		protocol = inq.protocol.String()
		m = fmt.Sprintf("%s %s",
			strings.TrimSpace(string(inq.vendor)),
			strings.TrimSpace(string(inq.product)))
	}

	return &Identity{
		Protocol:     protocol,
		Model:        m,
		Firmware:     strings.TrimSpace(string(inq.revision)),
		SerialNumber: strings.TrimSpace(string(inq.serial)),
	}, nil
}

func (d *scsiDrive) SerialNumber() ([]byte, error) {
	inq, err := d.inquire()
	if err != nil {
		return nil, err
	}
	return bytes.Clone(inq.serial), nil
}

func (d *scsiDrive) Close() error {
	return d.t.Close()
}

func (d *scsiDrive) send(cdb []byte, dir SCSIDirection, data []byte) error {
	if len(data) == 0 {
		return ErrEmptyBuffer
	}
	res, err := d.t.SendCDB(cdb, dir, data)
	if err != nil {
		return err
	}
	return scsiResultError(res)
}

// sendIn sends cdb, receiving length bytes.
func (d *scsiDrive) sendIn(cdb []byte, length int) ([]byte, error) {
	buf := make([]byte, length)
	if err := d.send(cdb, SCSIDataIn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// inquire reads the standard INQUIRY data, and the serial number and
// transport protocol from VPD pages, if supported. VPD pages that cannot be
// parsed are ignored, as the information they provide is optional.
func (d *scsiDrive) inquire() (*scsiInquiry, error) {
	if d.inq != nil {
		return d.inq, nil
	}

	buf, err := d.sendIn(scsiInquiryCDB(scsiInquiryLength), scsiInquiryLength)
	if err != nil {
		return nil, err
	}
	inq, err := parseSCSIInquiry(buf)
	if err != nil {
		return nil, err
	}

	buf, err = d.sendIn(scsiInquiryVPDCDB(scsiVPDSupportedPages, scsiVPDLength), scsiVPDLength)
	if err != nil {
		return nil, err
	}
	pages, _ := scsiVPDPage(buf, scsiVPDSupportedPages)

	if slices.Contains(pages, scsiVPDUnitSerialNumber) {
		buf, err = d.sendIn(scsiInquiryVPDCDB(scsiVPDUnitSerialNumber, scsiVPDLength), scsiVPDLength)
		if err != nil {
			return nil, err
		}
		if sn, err := scsiVPDPage(buf, scsiVPDUnitSerialNumber); err == nil {
			inq.serial = sn
		}
	}

	if slices.Contains(pages, scsiVPDDeviceIdentification) {
		buf, err = d.sendIn(scsiInquiryVPDCDB(scsiVPDDeviceIdentification, scsiDeviceIdentificationVPDLength), scsiDeviceIdentificationVPDLength)
		if err != nil {
			return nil, err
		}
		if did, err := scsiVPDPage(buf, scsiVPDDeviceIdentification); err == nil {
			inq.protocol = scsiDeviceIdentificationProtocol(did)
		}
	}

	d.inq = inq
	return inq, nil
}

// scsiResultError returns the error described by a SCSI command result,
// ErrNotSupported for the ILLEGAL REQUEST sense key.
func scsiResultError(res SCSIResult) error {
	if res.Status == scsiStatusGood {
		return nil
	}
	if key, ok := scsiSenseKey(res.Sense); ok {
		if key == scsiSenseKeyIllegalRequest {
			return ErrNotSupported
		}
		return fmt.Errorf("SCSI command failed: sense key %#02x", key)
	}
	return fmt.Errorf("SCSI command failed: status %#02x", res.Status)
}

// scsiSenseKey returns the sense key of fixed (0x70) or descriptor (0x72)
// format sense data, and false if the sense data is in neither format.
func scsiSenseKey(sense []byte) (uint8, bool) {
	if len(sense) < 1 {
		return 0, false
	}
	switch sense[0] & 0x7f {
	case 0x70:
		if len(sense) < 3 {
			return 0, false
		}
		return sense[2] & 0x0f, true
	case 0x72:
		if len(sense) < 2 {
			return 0, false
		}
		return sense[1] & 0x0f, true
	}
	return 0, false
}

func scsiInquiryCDB(allocLen uint16) []byte {
	cdb := make([]byte, 6)
	cdb[0] = scsiOpInquiry
	binary.BigEndian.PutUint16(cdb[3:], allocLen)
	return cdb
}

func scsiInquiryVPDCDB(page uint8, allocLen uint16) []byte {
	cdb := scsiInquiryCDB(allocLen)
	cdb[1] = 0x01 // EVPD
	cdb[2] = page
	return cdb
}

// scsiSecurityProtocolCDB returns a SECURITY PROTOCOL IN or OUT CDB,
// transferring length bytes, which must be a multiple of 512.
func scsiSecurityProtocolCDB(op, proto uint8, sps uint16, length int) ([]byte, error) {
	if length&0x1ff > 0 {
		return nil, fmt.Errorf("SECURITY PROTOCOL IN/OUT only supports 512-byte aligned transfers")
	}
	cdb := make([]byte, 12)
	cdb[0] = op
	cdb[1] = proto
	binary.BigEndian.PutUint16(cdb[2:], sps)
	// Seagate 7E200 series seems to require INC_512 to be set, and all other
	// drives tested seem to be fine with it, so we only support 512 byte
	// aligned transfers.
	cdb[4] = 1 << 7 // INC_512
	binary.BigEndian.PutUint32(cdb[6:], uint32(length/512))
	return cdb, nil
}

// parseSCSIInquiry parses standard INQUIRY data.
func parseSCSIInquiry(b []byte) (*scsiInquiry, error) {
	if len(b) < scsiInquiryLength {
		return nil, fmt.Errorf("%w: INQUIRY data too short (%d < %d)", ErrUnexpectedResponse, len(b), scsiInquiryLength)
	}

	// The length field counts the bytes after the 5 byte header, add it to get the full size
	if l := int(b[4]) + 5; l < scsiInquiryLength {
		return nil, fmt.Errorf("%w: length of INQUIRY data (%d < %d)", ErrUnexpectedResponse, l, scsiInquiryLength)
	}

	return &scsiInquiry{
		vendor:   b[8:16],
		product:  b[16:32],
		revision: b[32:36],
		protocol: scsiProtocolUnknown,
	}, nil
}

// scsiVPDPage validates the header of a VPD page, and returns its contents.
func scsiVPDPage(b []byte, page uint8) ([]byte, error) {
	if len(b) < 4 {
		return nil, fmt.Errorf("%w: VPD page too short", ErrUnexpectedResponse)
	}
	if b[1] != page {
		return nil, fmt.Errorf("%w: VPD page %#02x instead of %#02x", ErrUnexpectedResponse, b[1], page)
	}
	l := int(binary.BigEndian.Uint16(b[2:4])) // bytes after the 4 byte header
	if l == 0 || 4+l > len(b) {
		return nil, fmt.Errorf("%w: invalid VPD page %#02x length %d", ErrUnexpectedResponse, page, l)
	}
	return b[4 : 4+l], nil
}

// scsiProtocol is a SCSI transport protocol identifier.
type scsiProtocol int

const scsiProtocolUnknown scsiProtocol = -1

func (p scsiProtocol) String() string {
	switch p {
	case 0:
		return "FC"
	case 2:
		return "SSA-S3P"
	case 3:
		return "SBP"
	case 4:
		return "SRP"
	case 5:
		return "iSCSI"
	case 6:
		return "SAS"
	case 7:
		return "ADT"
	case 8:
		return "ACS"
	case 9:
		return "SCSI/USB"
	case 10:
		return "SCSI/PCIe"
	case 11:
		return "PCIe"
	default:
		return "SCSI/Unknown"
	}
}

// scsiDeviceIdentificationProtocol returns the transport protocol reported by
// the identification descriptors of the device identification VPD page, or
// scsiProtocolUnknown if none reports one.
func scsiDeviceIdentificationProtocol(did []byte) scsiProtocol {
	// We are only interested in the protocol identifier.
	// Loop through all ID descriptors and check for a valid protocol field.
	proto := scsiProtocolUnknown
	for len(did) >= 4 {
		l := int(did[3]) // bytes after the 4 byte descriptor header
		if l+4 > len(did) {
			break
		}
		part := did[:l+4]              // identifier descriptor
		piv := (part[1] & 0x80) > 0    // protocol identifier valid bit
		assoc := (part[1] & 0x30) >> 4 // association field
		if piv && assoc >= 1 && assoc <= 2 {
			proto = scsiProtocol(part[0] >> 4)
		}
		did = did[l+4:]
	}
	return proto
}
