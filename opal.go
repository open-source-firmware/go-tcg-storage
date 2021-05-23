// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package tcgstorage

import (
	"errors"
)

var (
	ErrNoOPAL20Support = errors.New("Device does not support OPAL 2.0")
)

type opalSession struct {
	d DriveIntf
}

func Open(d DriveIntf) (*opalSession, error) {
	// Ensure the device supports OPAL 2.0
	d0, err := Discovery0(d)
	if err != nil {
		return nil, err
	}
	if d0.OPAL20 == nil {
		return nil, ErrNoOPAL20Support
	}
	return &opalSession{d: d}, nil
}
