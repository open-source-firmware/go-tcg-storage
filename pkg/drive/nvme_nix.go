// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"unsafe"

	"github.com/dswarbrick/smart/ioctl"
	"golang.org/x/sys/unix"
)

const (
	NVME_ADMIN_IDENTIFY = 0x06
	NVME_SECURITY_SEND  = 0x81
	NVME_SECURITY_RECV  = 0x82
)

var NVME_IOCTL_ADMIN_CMD = ioctl.Iowr('N', 0x41, unsafe.Sizeof(nvmePassthruCommand{}))

// Defined in <linux/nvme_ioctl.h>
type nvmePassthruCommand struct {
	opcode       uint8
	flags        uint8  //nolint:structcheck,unused
	rsvd1        uint16 //nolint:structcheck,unused
	nsid         uint32
	cdw2         uint32 //nolint:structcheck,unused
	cdw3         uint32 //nolint:structcheck,unused
	metadata     uint64 //nolint:structcheck,unused
	addr         uint64
	metadata_len uint32 //nolint:structcheck,unused
	data_len     uint32
	cdw10        uint32
	cdw11        uint32 //nolint:structcheck,unused
	cdw12        uint32 //nolint:structcheck,unused
	cdw13        uint32 //nolint:structcheck,unused
	cdw14        uint32 //nolint:structcheck,unused
	cdw15        uint32 //nolint:structcheck,unused
	timeout_ms   uint32 //nolint:structcheck,unused
	result       uint32 //nolint:structcheck,unused
}

type nvmeAdminCommand nvmePassthruCommand

var errEmptyBuffer = errors.New("empty data buffer")

// Submits an admin command transferring data, and fails both on ioctl errors
// and on a non-zero NVMe completion status, which the kernel reports as a
// positive ioctl return value.
func nvmeAdminCmd(fd FdIntf, cmd *nvmeAdminCommand, data []byte) error {
	if len(data) == 0 {
		return errEmptyBuffer
	}

	// No function calls between taking the data address and the syscall, so
	// that the buffer cannot be moved in the meantime. unix.Syscall keeps cmd
	// itself in place, but not the buffer it points to, hence the KeepAlive.
	rawFd := fd.Fd()
	cmd.addr = uint64(uintptr(unsafe.Pointer(&data[0])))
	cmd.data_len = uint32(len(data))
	r1, _, errno := unix.Syscall(unix.SYS_IOCTL, rawFd, NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(cmd)))
	runtime.KeepAlive(data)
	runtime.KeepAlive(fd)
	if errno != 0 {
		return errno
	}
	if r1 != 0 {
		return nvmeStatusError(uint32(r1))
	}
	return nil
}

func nvmeStatusError(status uint32) error {
	sct := (status >> 8) & 0x7 // status code type
	sc := status & 0xff        // status code
	// Generic command status: Invalid Command Opcode, Invalid Field in Command
	if sct == 0 && (sc == 0x01 || sc == 0x02) {
		return ErrNotSupported
	}
	return fmt.Errorf("NVMe command failed: status code type %#x, status code %#02x", sct, sc)
}

type nvmeDrive struct {
	fd FdIntf
}

func (d *nvmeDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	cmd := nvmeAdminCommand{
		opcode: NVME_SECURITY_RECV,
		nsid:   0,
		cdw10:  uint32(proto&0xff)<<24 | uint32(sps)<<8,
		cdw11:  uint32(len(*data)),
	}
	return nvmeAdminCmd(d.fd, &cmd, *data)
}

func (d *nvmeDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	cmd := nvmeAdminCommand{
		opcode: NVME_SECURITY_SEND,
		nsid:   0,
		cdw10:  uint32(proto&0xff)<<24 | uint32(sps)<<8,
		cdw11:  uint32(len(data)),
	}
	return nvmeAdminCmd(d.fd, &cmd, data)
}

func (d *nvmeDrive) Identify() (*Identity, error) {
	i, err := identifyNvme(d.fd)
	if err != nil {
		return nil, err
	}
	return &Identity{
		Protocol:     "NVMe",
		Model:        strings.TrimSpace(string(i.ModelNumber[:])),
		SerialNumber: strings.TrimSpace(string(i.SerialNumber[:])),
		Firmware:     strings.TrimSpace(string(i.Firmware[:])),
	}, nil
}

func (d *nvmeDrive) SerialNumber() ([]byte, error) {
	i, err := identifyNvme(d.fd)
	if err != nil {
		return nil, err
	}
	return i.SerialNumber[:], nil
}

func (d *nvmeDrive) Close() error {
	return d.fd.Close()
}

func NVMEDrive(fd FdIntf) *nvmeDrive {
	// Save the full object reference to avoid the underlying File-like object
	// to be GC'd
	return &nvmeDrive{fd: fd}
}

type nvmeIdentity struct {
	_            uint16 /* Vid */
	_            uint16 /* Ssvid */
	SerialNumber [20]byte
	ModelNumber  [40]byte
	Firmware     [8]byte
}

func identifyNvme(fd FdIntf) (*nvmeIdentity, error) {
	raw := make([]byte, 4096)

	cmd := nvmeAdminCommand{
		opcode: NVME_ADMIN_IDENTIFY,
		nsid:   0, // Namespace 0, since we are identifying the controller
		cdw10:  1, // Identify controller
	}

	if err := nvmeAdminCmd(fd, &cmd, raw); err != nil {
		return nil, err
	}

	info := nvmeIdentity{}
	buf := bytes.NewBuffer(raw)
	// NVMe *seems* to use little endian, no experience though - but since we are
	// reading byte arrays it matters not.
	if err := binary.Read(buf, binary.LittleEndian, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func isNVME(f FdIntf) bool {
	i, err := identifyNvme(f)
	return err == nil && i != nil
}
