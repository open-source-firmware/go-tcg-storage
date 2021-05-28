// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"github.com/bluecmd/go-tcg-storage/pkg/core"
)

var (
	Base_TableTable    = TableUID{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	Base_MethodIDTable = TableUID{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00}
)

func Base_Method_IsSupported(s *core.Session, m core.MethodID) bool {
	mc := core.NewMethodCall(core.InvokingID(m), MethodIDGet)
	mc.StartList()
	mc.StartOptionalParameter(CellBlock_StartColumn)
	mc.UInt(Table_ColumnUID)
	mc.EndOptionalParameter()
	mc.StartOptionalParameter(CellBlock_EndColumn)
	mc.UInt(Table_ColumnUID)
	mc.EndOptionalParameter()
	mc.EndList()
	_, err := s.ExecuteMethod(mc)
	return err == nil
}
