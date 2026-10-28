// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"os"
)

// Open opens an NVMe, SCSI, SAS or SATA device node.
func Open(device string) (DriveIntf, error) {
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	// Probe the transports by identifying the device, which the drives cache
	if d := NewNVMeDrive(NewNVMeIoctlTransport(f)); identifies(d) {
		return d, nil
	}
	if d := NewSCSIDrive(NewSGIOTransport(f)); identifies(d) {
		return d, nil
	}

	if err := f.Close(); err != nil {
		return nil, err
	}
	return nil, ErrDeviceNotSupported
}

func identifies(d DriveIntf) bool {
	_, err := d.Identify()
	return err == nil
}
