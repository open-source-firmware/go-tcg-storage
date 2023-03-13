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
	"errors"
	"fmt"
	"unsafe"

	"github.com/dswarbrick/smart/ioctl"
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
	DEFAULT_TIMEOUT = 60000

	PIO_DATA_IN  = 4
	PIO_DATA_OUT = 5

	SENSE_ILLEGAL_REQUEST = 0x5

	DRIVER_SENSE = 0x8
)

var (
	ErrIllegalRequest = errors.New("illegal SCSI request")

	nativeEndian binary.ByteOrder
)

// SCSI CDB types
type (
	CDB6  [6]byte
	CDB10 [10]byte
	CDB12 [12]byte
	CDB16 [16]byte
)

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
	iovec_count     uint16       //nolint:structcheck,unused // 0 implies no scatter gather
	dxfer_len       uint32       // byte count of data transfer
	dxferp          uintptr      // points to data transfer memory or scatter gather list
	cmdp            uintptr      // points to command to perform
	sbp             uintptr      // points to sense_buffer memory
	timeout         uint32       // MAX_UINT -> no timeout (unit: millisec)
	flags           uint32       //nolint:structcheck,unused // 0 -> default, see SG_FLAG...
	pack_id         int32        //nolint:structcheck,unused // unused internally (normally)
	usr_ptr         uintptr      //nolint:structcheck,unused // unused internally
	status          uint8        // SCSI status
	masked_status   uint8        //nolint:structcheck,unused // shifted, masked scsi status
	msg_status      uint8        //nolint:structcheck,unused // messaging level data (optional)
	sb_len_wr       uint8        //nolint:structcheck,unused // byte count actually written to sbp
	host_status     uint16       // errors from host adapter
	driver_status   uint16       // errors from software driver
	resid           int32        //nolint:structcheck,unused // dxfer_len - actual_transferred
	duration        uint32       //nolint:structcheck,unused // time taken by cmd (unit: millisec)
	info            uint32       // auxiliary information
}

func execGenericIO(fd uintptr, hdr *sgIoHdr, sense []byte) error {
	if err := ioctl.Ioctl(fd, SG_IO, uintptr(unsafe.Pointer(hdr))); err != nil {
		return err
	}

	// See http://www.t10.org/lists/2status.htm for SCSI status codes
	if hdr.info&SG_INFO_OK_MASK != SG_INFO_OK {
		if hdr.driver_status == DRIVER_SENSE {
			if sense[0]&0x7f == 0x70 {
				if sense[2]&0x0f == SENSE_ILLEGAL_REQUEST {
					return ErrIllegalRequest
				}
				return fmt.Errorf("SCSI status: sense key: %#02x", sense[2]&0x0f)
			}
			if sense[0]&0x7f == 0x72 {
				if sense[1]&0x0f == SENSE_ILLEGAL_REQUEST {
					return ErrIllegalRequest
				}
				return fmt.Errorf("SCSI status: sense key: %#02x", sense[1]&0x0f)
			}
		}
		return fmt.Errorf("SCSI status: %#02x, host status: %#02x, driver status: %#02x, response: %#02x",
			hdr.status, hdr.host_status, hdr.driver_status, sense[0])
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

	return execGenericIO(fd, &hdr, senseBuf)
}
