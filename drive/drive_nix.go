// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"fmt"
	"os"
)

func Open(device string) (driveIntf, error) {
	d, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	// Note that SATA implements part of the SCSI standard, so try it before SCSI
	if isNVME(d) {
		return nil, fmt.Errorf("NVMe devices not implemented")
	} else if isATA(d) {
		return nil, fmt.Errorf("ATA devices not implemented")
	} else if isSCSI(d) {
		return nil, fmt.Errorf("SCSI devices not implemented")
	}

	return nil, fmt.Errorf("Device type not supported")
}
