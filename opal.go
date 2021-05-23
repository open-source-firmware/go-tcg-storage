// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package tcgstorage

import (
	"errors"
)

var (
	ErrNoOpalV2Support = errors.New("Device does not support Opal 2.0")
)

type opalSession struct {
	d DriveIntf
}

func OpalSession(d DriveIntf) (*opalSession, error) {
	// Ensure the device supports OPAL 2.0
	d0, err := Discovery0(d)
	if err != nil {
		return nil, err
	}
	if d0.OpalV2 == nil {
		return nil, ErrNoOpalV2Support
	}
	return &opalSession{d: d}, nil
}
