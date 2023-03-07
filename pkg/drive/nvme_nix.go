// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"strings"
	"unsafe"

	"github.com/dswarbrick/smart/ioctl"
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

type nvmeDrive struct {
	fd FdIntf
}

func (d *nvmeDrive) IFRecv(proto SecurityProtocol, sps uint16, data *[]byte) error {
	cmd := nvmeAdminCommand{
		opcode:   NVME_SECURITY_RECV,
		nsid:     0,
		addr:     uint64(uintptr(unsafe.Pointer(&(*data)[0]))),
		data_len: uint32(len(*data)),
		cdw10:    uint32(proto&0xff)<<24 | uint32(sps)<<8,
		cdw11:    uint32(len(*data)),
	}

	err := ioctl.Ioctl(d.fd.Fd(), NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd)))
	runtime.KeepAlive(d.fd)
	return err
}

func (d *nvmeDrive) IFSend(proto SecurityProtocol, sps uint16, data []byte) error {
	cmd := nvmeAdminCommand{
		opcode:   NVME_SECURITY_SEND,
		nsid:     0,
		addr:     uint64(uintptr(unsafe.Pointer(&data[0]))),
		data_len: uint32(len(data)),
		cdw10:    uint32(proto&0xff)<<24 | uint32(sps)<<8,
		cdw11:    uint32(len(data)),
	}

	err := ioctl.Ioctl(d.fd.Fd(), NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd)))
	runtime.KeepAlive(d.fd)
	return err
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

	cmd := nvmePassthruCommand{
		opcode:   NVME_ADMIN_IDENTIFY,
		nsid:     0, // Namespace 0, since we are identifying the controller
		addr:     uint64(uintptr(unsafe.Pointer(&raw[0]))),
		data_len: uint32(len(raw)),
		cdw10:    1, // Identify controller
	}

	// TODO: Replace with https://go-review.googlesource.com/c/sys/+/318210/ if accepted
	err := ioctl.Ioctl(fd.Fd(), NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd)))
	runtime.KeepAlive(fd)
	if err != nil {
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
