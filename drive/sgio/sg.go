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

// SCSI generic IO functions.

package sgio

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/bluecmd/go-opal/drive/ioctl"
)

type CDBDirection int32

const (
	CDBToDevice     CDBDirection = -2
	CDBFromDevice   CDBDirection = -3
	CDBToFromDevice CDBDirection = -4

	SG_INFO_OK_MASK = 0x1
	SG_INFO_OK      = 0x0

	SG_IO = 0x2285

	// Timeout in milliseconds
	DEFAULT_TIMEOUT = 20000

	PIO_DATA_IN  = 4
	PIO_DATA_OUT = 5
)

var (
	nativeEndian binary.ByteOrder
)

// SCSI CDB types
type CDB6 [6]byte
type CDB10 [10]byte
type CDB12 [12]byte
type CDB16 [16]byte

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
	interface_id    int32        // 'S' for SCSI generic (required)
	dxfer_direction CDBDirection // data transfer direction
	cmd_len         uint8        // SCSI command length (<= 16 bytes)
	mx_sb_len       uint8        // max length to write to sbp
	iovec_count     uint16       // 0 implies no scatter gather
	dxfer_len       uint32       // byte count of data transfer
	dxferp          uintptr      // points to data transfer memory or scatter gather list
	cmdp            uintptr      // points to command to perform
	sbp             uintptr      // points to sense_buffer memory
	timeout         uint32       // MAX_UINT -> no timeout (unit: millisec)
	flags           uint32       // 0 -> default, see SG_FLAG...
	pack_id         int32        // unused internally (normally)
	usr_ptr         uintptr      // unused internally
	status          uint8        // SCSI status
	masked_status   uint8        // shifted, masked scsi status
	msg_status      uint8        // messaging level data (optional)
	sb_len_wr       uint8        // byte count actually written to sbp
	host_status     uint16       // errors from host adapter
	driver_status   uint16       // errors from software driver
	resid           int32        // dxfer_len - actual_transferred
	duration        uint32       // time taken by cmd (unit: millisec)
	info            uint32       // auxiliary information
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

func SendCDB(fd uintptr, cdb []byte, dir CDBDirection, buf *[]byte) error {
	senseBuf := make([]byte, 32)

	hdr := sgIoHdr{
		interface_id:    'S',
		dxfer_direction: dir,
		timeout:         DEFAULT_TIMEOUT,
		cmd_len:         uint8(len(cdb)),
		mx_sb_len:       uint8(len(senseBuf)),
		dxfer_len:       uint32(len(*buf)),
		dxferp:          uintptr(unsafe.Pointer(&(*buf)[0])),
		cmdp:            uintptr(unsafe.Pointer(&cdb[0])),
		sbp:             uintptr(unsafe.Pointer(&senseBuf[0])),
	}

	return execGenericIO(fd, &hdr)
}
