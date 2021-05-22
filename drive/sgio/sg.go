// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
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

// SCSI generic IO functions.

package sgio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/bluecmd/go-opal/drive/ioctl"
)

const (
	SG_DXFER_NONE        = -1
	SG_DXFER_TO_DEV      = -2
	SG_DXFER_FROM_DEV    = -3
	SG_DXFER_TO_FROM_DEV = -4

	SG_INFO_OK_MASK = 0x1
	SG_INFO_OK      = 0x0

	SG_IO = 0x2285

	// Timeout in milliseconds
	DEFAULT_TIMEOUT = 20000
)

var (
	nativeEndian binary.ByteOrder
)




const (
	// SCSI commands used by this package
	SCSI_INQUIRY          = 0x12
	SCSI_MODE_SENSE_6     = 0x1a
	SCSI_READ_CAPACITY_10 = 0x25
	SCSI_ATA_PASSTHRU_16  = 0x85

	// Minimum length of standard INQUIRY response
	INQ_REPLY_LEN = 36

	// SCSI-3 mode pages
	RIGID_DISK_DRIVE_GEOMETRY_PAGE = 0x04

	// Mode page control field
	MPAGE_CONTROL_DEFAULT = 2
)

// SCSI CDB types
type CDB6 [6]byte
type CDB10 [10]byte
type CDB16 [16]byte

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
	return fmt.Sprintf("%.8s  %.16s  %.4s", inq.VendorIdent, inq.ProductIdent, inq.ProductRev)
}

// Determine native endianness of system
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}

// SCSI generic ioctl header, defined as sg_io_hdr_t in <scsi/sg.h>
type sgIoHdr struct {
	interface_id    int32   // 'S' for SCSI generic (required)
	dxfer_direction int32   // data transfer direction
	cmd_len         uint8   // SCSI command length (<= 16 bytes)
	mx_sb_len       uint8   // max length to write to sbp
	iovec_count     uint16  // 0 implies no scatter gather
	dxfer_len       uint32  // byte count of data transfer
	dxferp          uintptr // points to data transfer memory or scatter gather list
	cmdp            uintptr // points to command to perform
	sbp             uintptr // points to sense_buffer memory
	timeout         uint32  // MAX_UINT -> no timeout (unit: millisec)
	flags           uint32  // 0 -> default, see SG_FLAG...
	pack_id         int32   // unused internally (normally)
	usr_ptr         uintptr // unused internally
	status          uint8   // SCSI status
	masked_status   uint8   // shifted, masked scsi status
	msg_status      uint8   // messaging level data (optional)
	sb_len_wr       uint8   // byte count actually written to sbp
	host_status     uint16  // errors from host adapter
	driver_status   uint16  // errors from software driver
	resid           int32   // dxfer_len - actual_transferred
	duration        uint32  // time taken by cmd (unit: millisec)
	info            uint32  // auxiliary information
}

type sgioError struct {
	scsiStatus   uint8
	hostStatus   uint16
	driverStatus uint16
	senseBuf     [32]byte // FIXME: This is not yet populated by anything
}

func (e sgioError) Error() string {
	return fmt.Sprintf("SCSI status: %#02x, host status: %#02x, driver status: %#02x",
		e.scsiStatus, e.hostStatus, e.driverStatus)
}

func execGenericIO(fd uintptr, hdr *sgIoHdr) error {
	if err := ioctl.Ioctl(fd, SG_IO, uintptr(unsafe.Pointer(hdr))); err != nil {
		return err
	}

	// See http://www.t10.org/lists/2status.htm for SCSI status codes
	if hdr.info&SG_INFO_OK_MASK != SG_INFO_OK {
		err := sgioError{
			scsiStatus:   hdr.status,
			hostStatus:   hdr.host_status,
			driverStatus: hdr.driver_status,
		}
		return err
	}

	return nil
}

// inquiry sends a SCSI INQUIRY command to a device and returns an InquiryResponse struct.
// TODO: Add support for Vital Product Data (VPD)
func Inquiry(fd uintptr) (InquiryResponse, error) {
	var resp InquiryResponse

	respBuf := make([]byte, INQ_REPLY_LEN)

	cdb := CDB6{SCSI_INQUIRY}
	binary.BigEndian.PutUint16(cdb[3:], uint16(len(respBuf)))

	if err := SendCDB(fd, cdb[:], &respBuf); err != nil {
		return resp, err
	}

	binary.Read(bytes.NewBuffer(respBuf), nativeEndian, &resp)

	return resp, nil
}

// SendCDB sends a SCSI Command Descriptor Block to the device and writes the response into the
// supplied []byte pointer.
// TODO: Return SCSI status code, sense buf etc as part of error
func SendCDB(fd uintptr, cdb []byte, respBuf *[]byte) error {
	senseBuf := make([]byte, 32)

	// Populate required fields of "sg_io_hdr_t" struct
	hdr := sgIoHdr{
		interface_id:    'S',
		dxfer_direction: SG_DXFER_FROM_DEV,
		timeout:         DEFAULT_TIMEOUT,
		cmd_len:         uint8(len(cdb)),
		mx_sb_len:       uint8(len(senseBuf)),
		dxfer_len:       uint32(len(*respBuf)),
		dxferp:          uintptr(unsafe.Pointer(&(*respBuf)[0])),
		cmdp:            uintptr(unsafe.Pointer(&cdb[0])),
		sbp:             uintptr(unsafe.Pointer(&senseBuf[0])),
	}

	return execGenericIO(fd, &hdr)
}

// modeSense sends a SCSI MODE SENSE(6) command to a device.
func ModeSense(fd uintptr, pageNum, subPageNum, pageControl uint8) ([]byte, error) {
	respBuf := make([]byte, 64)

	cdb := CDB6{SCSI_MODE_SENSE_6}
	cdb[2] = (pageControl << 6) | (pageNum & 0x3f)
	cdb[3] = subPageNum
	cdb[4] = uint8(len(respBuf))

	if err := SendCDB(fd, cdb[:], &respBuf); err != nil {
		return respBuf, err
	}

	return respBuf, nil
}

// readCapacity sends a SCSI READ CAPACITY(10) command to a device and returns the capacity in bytes.
func ReadCapacity(fd uintptr) (uint64, error) {
	respBuf := make([]byte, 8)
	cdb := CDB10{SCSI_READ_CAPACITY_10}

	if err := SendCDB(fd, cdb[:], &respBuf); err != nil {
		return 0, err
	}

	lastLBA := binary.BigEndian.Uint32(respBuf[0:]) // max. addressable LBA
	LBsize := binary.BigEndian.Uint32(respBuf[4:])  // logical block (i.e., sector) size
	capacity := (uint64(lastLBA) + 1) * uint64(LBsize)

	return capacity, nil
}
