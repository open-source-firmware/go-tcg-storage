// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"unsafe"

	"github.com/bluecmd/go-opal/drive/ioctl"
)

const (
	NVME_ADMIN_IDENTIFY = 0x06
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

type nvmeDrive struct {
}

func (d *nvmeDrive) SecurityCommand() {

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
