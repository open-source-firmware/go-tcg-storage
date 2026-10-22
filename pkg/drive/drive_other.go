// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !linux

package drive

// Open is not supported on this platform. Drives can be created with
// NewSCSIDrive or NewNVMeDrive from a caller provided transport instead, or be
// a caller implemented DriveIntf.
func Open(device string) (DriveIntf, error) {
	return nil, ErrDeviceNotSupported
}
