// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Copyright 2021 Christian Svensson. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sgio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	ATA_PASSTHROUGH     = 0xa1
	ATA_TRUSTED_RCV     = 0x5c
	ATA_TRUSTED_SND     = 0x5e
	ATA_IDENTIFY_DEVICE = 0xec

	SCSI_INQUIRY          = 0x12
	SCSI_MODE_SENSE_6     = 0x1a
	SCSI_READ_CAPACITY_10 = 0x25
	SCSI_ATA_PASSTHRU_16  = 0x85
	SCSI_SECURITY_IN      = 0xa2
	SCSI_SECURITY_OUT     = 0xb5
)

// SCSI INQUIRY response
type InquiryResponse struct {
	Peripheral   byte // peripheral qualifier, device type
	_            byte
	Version      byte
	_            [5]byte
	VendorIdent  [8]byte
	ProductIdent [16]byte
	ProductRev   [4]byte
}

func (inq InquiryResponse) String() string {
	return fmt.Sprintf("Type=0x%x, Vendor=%s, Product=%s, Revision=%s",
		inq.Peripheral,
		strings.TrimSpace(string(inq.VendorIdent[:])),
		strings.TrimSpace(string(inq.ProductIdent[:])),
		strings.TrimSpace(string(inq.ProductRev[:])))
}

// ATA IDENTFY DEVICE response
type IdentifyDeviceResponse struct {
	_        [20]byte
	Serial   [20]byte
	_        [6]byte
	Firmware [8]byte
	Model    [40]byte
	_        [418]byte
}

func ATAString(b []byte) string {
	out := make([]byte, len(b))
	for i := 0; i < len(b)/2; i++ {
		out[i*2] = b[i*2+1]
		out[i*2+1] = b[i*2]
	}
	return string(out)
}

func (id IdentifyDeviceResponse) String() string {
	return fmt.Sprintf("Serial=%s, Firmware=%s, Model=%s",
		strings.TrimSpace(ATAString(id.Serial[:])),
		strings.TrimSpace(ATAString(id.Firmware[:])),
		strings.TrimSpace(ATAString(id.Model[:])))
}

// INQUIRY - Returns parsed inquiry data.
func SCSIInquiry(fd uintptr) (InquiryResponse, error) {
	var resp InquiryResponse

	respBuf := make([]byte, 36)

	cdb := CDB6{SCSI_INQUIRY}
	binary.BigEndian.PutUint16(cdb[3:], uint16(len(respBuf)))

	if err := SendCDB(fd, cdb[:], CDBFromDevice, &respBuf); err != nil {
		return resp, err
	}

	binary.Read(bytes.NewBuffer(respBuf), nativeEndian, &resp)

	return resp, nil
}

// ATA Passthrough via SCSI (which is what Linux uses for all ATA these days)
func ATAIdentify(fd uintptr) (IdentifyDeviceResponse, error) {
	var resp IdentifyDeviceResponse

	respBuf := make([]byte, 512)

	cdb := CDB12{ATA_PASSTHROUGH}
	cdb[1] = PIO_DATA_IN << 1
	cdb[2] = 0x0E
	cdb[4] = 1
	cdb[9] = ATA_IDENTIFY_DEVICE

	if err := SendCDB(fd, cdb[:], CDBFromDevice, &respBuf); err != nil {
		return resp, err
	}

	binary.Read(bytes.NewBuffer(respBuf), nativeEndian, &resp)

	return resp, nil
}

// SCSI MODE SENSE(6) - Returns the raw response
func SCSIModeSense(fd uintptr, pageNum, subPageNum, pageControl uint8) ([]byte, error) {
	respBuf := make([]byte, 64)

	cdb := CDB6{SCSI_MODE_SENSE_6}
	cdb[2] = (pageControl << 6) | (pageNum & 0x3f)
	cdb[3] = subPageNum
	cdb[4] = uint8(len(respBuf))

	if err := SendCDB(fd, cdb[:], CDBFromDevice, &respBuf); err != nil {
		return respBuf, err
	}

	return respBuf, nil
}

// SCSI READ CAPACITY(10) - Returns the capacity in bytes
func SCSIReadCapacity(fd uintptr) (uint64, error) {
	respBuf := make([]byte, 8)
	cdb := CDB10{SCSI_READ_CAPACITY_10}

	if err := SendCDB(fd, cdb[:], CDBFromDevice, &respBuf); err != nil {
		return 0, err
	}

	lastLBA := binary.BigEndian.Uint32(respBuf[0:]) // max. addressable LBA
	LBsize := binary.BigEndian.Uint32(respBuf[4:])  // logical block (i.e., sector) size
	capacity := (uint64(lastLBA) + 1) * uint64(LBsize)

	return capacity, nil
}

// ATA TRUSTED RECEIVE
func ATATrustedReceive(fd uintptr, proto uint8, comID uint16, resp *[]byte) error {
	cdb := CDB12{ATA_PASSTHROUGH}
	cdb[1] = PIO_DATA_IN << 1
	cdb[2] = 0x0E
	cdb[3] = proto
	cdb[4] = uint8(len(*resp) / 512)
	cdb[6] = uint8(comID & 0xff)
	cdb[7] = uint8((comID & 0xff00) >> 8)
	cdb[9] = ATA_TRUSTED_RCV
	if err := SendCDB(fd, cdb[:], CDBFromDevice, resp); err != nil {
		return err
	}
	return nil
}

// ATA TRUSTED SEND
func ATATrustedSend(fd uintptr, proto uint8, comID uint16, in []byte) error {
	cdb := CDB12{ATA_PASSTHROUGH}
	cdb[1] = PIO_DATA_OUT << 1
	cdb[2] = 0x06
	cdb[3] = proto
	cdb[4] = uint8(len(in) / 512)
	cdb[6] = uint8(comID & 0xff)
	cdb[7] = uint8((comID & 0xff00) >> 8)
	cdb[9] = ATA_TRUSTED_RCV
	if err := SendCDB(fd, cdb[:], CDBToDevice, &in); err != nil {
		return err
	}
	return nil
}

// SCSI SECURITY IN
func SCSISecurityIn(fd uintptr, proto uint8, sps uint16, resp *[]byte) error {
	if len(*resp)%512 > 0 {
		return fmt.Errorf("SCSISecurityIn only supports 512-byte aligned buffers")
	}
	cdb := CDB12{SCSI_SECURITY_IN}
	cdb[1] = proto
	cdb[2] = uint8((sps & 0xff00) >> 8)
	cdb[3] = uint8(sps & 0xff)
	//
	// Seagate 7E200 series seems to require INC_512 to be set, and all other
	// drives tested seem to be fine with it, so we only support 512 byte aligned
	cdb[4] = 1 << 7 // INC_512 = 1
	binary.BigEndian.PutUint32(cdb[6:], uint32(len(*resp)/512))

	if err := SendCDB(fd, cdb[:], CDBFromDevice, resp); err != nil {
		return err
	}
	return nil
}

// SCSI SECURITY OUT
func SCSISecurityOut(fd uintptr, proto uint8, sps uint16, in []byte) error {
	if len(in)%512 > 0 {
		return fmt.Errorf("SCSISecurityOut only supports 512-byte aligned buffers")
	}
	cdb := CDB12{SCSI_SECURITY_OUT}
	cdb[1] = proto
	cdb[2] = uint8((sps & 0xff00) >> 8)
	cdb[3] = uint8(sps & 0xff)
	//
	// Seagate 7E200 series seems to require INC_512 to be set, and all other
	// drives tested seem to be fine with it, so we only support 512 byte aligned
	// buffers.
	cdb[4] = 1 << 7 // INC_512 = 1
	binary.BigEndian.PutUint32(cdb[6:], uint32(len(in)/512))

	if err := SendCDB(fd, cdb[:], CDBToDevice, &in); err != nil {
		return err
	}
	return nil
}
