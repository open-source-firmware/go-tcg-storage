// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"
	"unsafe"

	"github.com/bluecmd/go-opal/drive/ioctl"
)

const (
	NVME_ADMIN_IDENTIFY = 0x06
	NVME_SECURITY_SEND  = 0x81
	NVME_SECURITY_RECV  = 0x82
)

var (
	NVME_IOCTL_ADMIN_CMD = ioctl.Iowr('N', 0x41, unsafe.Sizeof(nvmePassthruCommand{}))
)

// Defined in <linux/nvme_ioctl.h>
type nvmePassthruCommand struct {
	opcode       uint8
	flags        uint8
	rsvd1        uint16
	nsid         uint32
	cdw2         uint32
	cdw3         uint32
	metadata     uint64
	addr         uint64
	metadata_len uint32
	data_len     uint32
	cdw10        uint32
	cdw11        uint32
	cdw12        uint32
	cdw13        uint32
	cdw14        uint32
	cdw15        uint32
	timeout_ms   uint32
	result       uint32
}

type nvmeAdminCommand nvmePassthruCommand

type nvmeDrive struct {
	fd uintptr
}

func (d *nvmeDrive) IFRecv(proto SecurityProtocol, comID ComID, data *[]byte) error {
	cmd := nvmeAdminCommand{
		opcode:   NVME_SECURITY_RECV,
		nsid:     0,
		addr:     uint64(uintptr(unsafe.Pointer(&(*data)[0]))),
		data_len: uint32(len(*data)),
		cdw10:    uint32(proto&0xff)<<24 | uint32(comID&0xffff)<<8,
		cdw11:    uint32(len(*data)),
	}

	return ioctl.Ioctl(d.fd, NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd)))
}

func (d *nvmeDrive) IFSend(proto SecurityProtocol, comID ComID, dnvme *[]byte) error {
	return nil
}

func (d *nvmeDrive) Close() error {
	return os.NewFile(d.fd, "").Close()
}

func NVMEDrive(fd FdIntf) *nvmeDrive {
	return &nvmeDrive{fd: fd.Fd()}
}

func isNVME(f FdIntf) bool {
	buf := make([]byte, 4096)

	cmd := nvmePassthruCommand{
		opcode:   NVME_ADMIN_IDENTIFY,
		nsid:     0, // Namespace 0, since we are identifying the controller
		addr:     uint64(uintptr(unsafe.Pointer(&buf[0]))),
		data_len: uint32(len(buf)),
		cdw10:    1, // Identify controller
	}

	// TODO: Replace with https://go-review.googlesource.com/c/sys/+/318210/ if accepted
	err := ioctl.Ioctl(f.Fd(), NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd)))
	return err == nil
}
