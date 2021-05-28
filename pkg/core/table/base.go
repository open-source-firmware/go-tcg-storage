// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

var (
	Base_TableTable         = TableUID{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	Base_MethodIDTable      = TableUID{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00}
	Base_AccessControlTable = TableUID{0x00, 0x00, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00}
)
