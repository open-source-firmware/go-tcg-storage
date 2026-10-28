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
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// Data transfer directions, defined as SG_DXFER_* in <scsi/sg.h>
	sgDxferToDev   = -2
	sgDxferFromDev = -3

	sgInfoOkMask = 0x1
	sgInfoOk     = 0x0

	sgIO = 0x2285

	// Timeout in milliseconds
	sgTimeout = 60000

	sgDriverSense = 0x8

	sgSenseLength = 32
)

// SCSI generic ioctl header, defined as sg_io_hdr_t in <scsi/sg.h>
type sgIoHdr struct {
	interface_id    int32   // 'S' for SCSI generic (required)
	dxfer_direction int32   // data transfer direction
	cmd_len         uint8   // SCSI command length (<= 16 bytes)
	mx_sb_len       uint8   // max length to write to sbp
	iovec_count     uint16  //nolint:structcheck,unused // 0 implies no scatter gather
	dxfer_len       uint32  // byte count of data transfer
	dxferp          uintptr // points to data transfer memory or scatter gather list
	cmdp            uintptr // points to command to perform
	sbp             uintptr // points to sense_buffer memory
	timeout         uint32  // MAX_UINT -> no timeout (unit: millisec)
	flags           uint32  //nolint:structcheck,unused // 0 -> default, see SG_FLAG...
	pack_id         int32   //nolint:structcheck,unused // unused internally (normally)
	usr_ptr         uintptr //nolint:structcheck,unused // unused internally
	status          uint8   // SCSI status
	masked_status   uint8   //nolint:structcheck,unused // shifted, masked scsi status
	msg_status      uint8   //nolint:structcheck,unused // messaging level data (optional)
	sb_len_wr       uint8   // byte count actually written to sbp
	host_status     uint16  // errors from host adapter
	driver_status   uint16  // errors from software driver
	resid           int32   //nolint:structcheck,unused // dxfer_len - actual_transferred
	duration        uint32  //nolint:structcheck,unused // time taken by cmd (unit: millisec)
	info            uint32  // auxiliary information
}

// SGIOTransport is a SCSITransport sending commands through the Linux SCSI
// generic (SG_IO) ioctl, which SCSI, SAS and SATA device nodes support.
type SGIOTransport struct {
	f *os.File
}

// NewSGIOTransport returns a transport for an opened device node, which it
// takes ownership of.
func NewSGIOTransport(f *os.File) *SGIOTransport {
	return &SGIOTransport{f: f}
}

func (t *SGIOTransport) SendCDB(cdb []byte, dir SCSIDirection, data []byte) (SCSIResult, error) {
	if len(cdb) == 0 || len(data) == 0 {
		return SCSIResult{}, ErrEmptyBuffer
	}

	var sgDir int32
	switch dir {
	case SCSIDataIn:
		sgDir = sgDxferFromDev
	case SCSIDataOut:
		sgDir = sgDxferToDev
	default:
		return SCSIResult{}, fmt.Errorf("unknown SCSI data direction %d", dir)
	}

	fd := t.f.Fd()
	sense := make([]byte, sgSenseLength)

	// No function calls between taking the buffer addresses and the syscall,
	// so that the buffers cannot be moved in the meantime. unix.Syscall keeps
	// hdr itself in place, but not the buffers it points to, hence the
	// KeepAlives.
	hdr := sgIoHdr{
		interface_id:    'S',
		dxfer_direction: sgDir,
		timeout:         sgTimeout,
		cmd_len:         uint8(len(cdb)),
		mx_sb_len:       uint8(len(sense)),
		dxfer_len:       uint32(len(data)),
		dxferp:          uintptr(unsafe.Pointer(&data[0])),
		cmdp:            uintptr(unsafe.Pointer(&cdb[0])),
		sbp:             uintptr(unsafe.Pointer(&sense[0])),
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, sgIO, uintptr(unsafe.Pointer(&hdr)))
	runtime.KeepAlive(cdb)
	runtime.KeepAlive(data)
	runtime.KeepAlive(sense)
	runtime.KeepAlive(t.f)
	if errno != 0 {
		return SCSIResult{}, errno
	}

	if hdr.info&sgInfoOkMask == sgInfoOk {
		return SCSIResult{Status: scsiStatusGood}, nil
	}
	// See http://www.t10.org/lists/2status.htm for SCSI status codes
	if hdr.host_status != 0 || hdr.driver_status&^sgDriverSense != 0 {
		return SCSIResult{}, fmt.Errorf("SG_IO failed: SCSI status: %#02x, host status: %#02x, driver status: %#02x",
			hdr.status, hdr.host_status, hdr.driver_status)
	}
	res := SCSIResult{Status: hdr.status}
	if hdr.driver_status&sgDriverSense != 0 {
		res.Sense = sense
		if n := int(hdr.sb_len_wr); n > 0 && n < len(sense) {
			res.Sense = sense[:n]
		}
	}
	return res, nil
}

func (t *SGIOTransport) Close() error {
	return t.f.Close()
}
