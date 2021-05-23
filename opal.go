// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package tcgstorage

import (
	"errors"
)

var (
	ErrNoOpalV2Support = errors.New("device does not support Opal 2.0")
)

