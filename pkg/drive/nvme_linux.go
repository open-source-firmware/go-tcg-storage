// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"
	"runtime"
	"unsafe"

	"github.com/dswarbrick/smart/ioctl"
	"golang.org/x/sys/unix"
)

var nvmeIoctlAdminCmd = ioctl.Iowr('N', 0x41, unsafe.Sizeof(nvmePassthruCommand{}))

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
	cdw11        uint32
	cdw12        uint32
	cdw13        uint32
	cdw14        uint32
	cdw15        uint32
	timeout_ms   uint32 //nolint:structcheck,unused
	result       uint32 //nolint:structcheck,unused
}

// NVMeIoctlTransport is an NVMeTransport sending admin commands through the
// Linux NVMe driver (NVME_IOCTL_ADMIN_CMD), which NVMe controller and
// namespace device nodes support.
type NVMeIoctlTransport struct {
	f *os.File
}

// NewNVMeIoctlTransport returns a transport for an opened device node, which
// it takes ownership of.
func NewNVMeIoctlTransport(f *os.File) *NVMeIoctlTransport {
	return &NVMeIoctlTransport{f: f}
}

// AdminCommand submits an admin command transferring data. The kernel reports
// a non-zero completion status as a positive ioctl return value.
func (t *NVMeIoctlTransport) AdminCommand(c NVMeAdminCommand, data []byte) (NVMeStatus, error) {
	if len(data) == 0 {
		return 0, ErrEmptyBuffer
	}

	cmd := nvmePassthruCommand{
		opcode: c.Opcode,
		nsid:   c.NSID,
		cdw10:  c.CDW10,
		cdw11:  c.CDW11,
		cdw12:  c.CDW12,
		cdw13:  c.CDW13,
		cdw14:  c.CDW14,
		cdw15:  c.CDW15,
	}

	// No function calls between taking the data address and the syscall, so
	// that the buffer cannot be moved in the meantime. unix.Syscall keeps cmd
	// itself in place, but not the buffer it points to, hence the KeepAlive.
	fd := t.f.Fd()
	cmd.addr = uint64(uintptr(unsafe.Pointer(&data[0])))
	cmd.data_len = uint32(len(data))
	r1, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, nvmeIoctlAdminCmd, uintptr(unsafe.Pointer(&cmd)))
	runtime.KeepAlive(data)
	runtime.KeepAlive(t.f)
	if errno != 0 {
		return 0, errno
	}
	return NVMeStatus(r1), nil
}

func (t *NVMeIoctlTransport) Close() error {
	return t.f.Close()
}
