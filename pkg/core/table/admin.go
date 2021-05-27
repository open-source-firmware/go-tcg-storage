// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/stream"
)

var (
	Admin_TPerInfoTable = TableUID{0x00, 0x00, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00}
	Admin_C_PINTable    = TableUID{0x00, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00, 0x00}

	Admin_C_PIN_ColumnPIN uint = 3

	// TODO: This is taken from the Opal spec, not sure how to dynamically find it...
	Admin_TPerInfoObj   RowUID = Admin_TPerInfoTable.Row([4]byte{0x00, 0x03, 0x00, 0x01})
	Admin_C_PIN_MSIDRow RowUID = Admin_C_PINTable.Row([4]byte{0x00, 0x00, 0x84, 0x02})
)

func Admin_C_PIN_MSID_GetPIN(s *core.Session) ([]byte, error) {
	mc := core.NewMethodCall(core.InvokingID(Admin_C_PIN_MSIDRow), MethodIDGet)
	mc.StartList()
	mc.StartOptionalParameter(CellBlock_StartColumn)
	mc.UInt(Admin_C_PIN_ColumnPIN)
	mc.EndOptionalParameter()
	mc.StartOptionalParameter(CellBlock_EndColumn)
	mc.UInt(Admin_C_PIN_ColumnPIN)
	mc.EndOptionalParameter()
	mc.EndList()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	val, err := parseGetResult(resp)
	if err != nil {
		return nil, err
	}
	raw, ok := val[Admin_C_PIN_ColumnPIN]
	if !ok {
		return nil, fmt.Errorf("no PIN column in result")
	}
	pin, ok := raw.([]byte)
	if !ok {
		return nil, fmt.Errorf("malformed PIN column")
	}
	return pin, nil
}

type Admin_TPerInfoRow struct {
	UID                     RowUID
	Bytes                   *uint64
	GUDID                   *[12]byte
	Generation              *uint32
	FirmwareVersion         *uint32
	ProtocolVersion         *uint32
	SpaceForIssuance        *uint64
	SSC                     []string
	ProgrammaticResetEnable *bool
}

func Admin_TPerInfo(s *core.Session) (map[RowUID]Admin_TPerInfoRow, error) {
	res := map[RowUID]Admin_TPerInfoRow{}
	mc := core.NewMethodCall(core.InvokingID(Admin_TPerInfoObj), MethodIDGet)
	mc.StartList()
	mc.EndList()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	val, err := parseGetResult(resp)
	if err != nil {
		return nil, err
	}

	row := Admin_TPerInfoRow{}
	for col, val := range val {
		switch col {
		case 0:
			v, ok := val.([]byte)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			copy(row.UID[:], v[:8])
		case 1:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			row.Bytes = &vv
		case 2:
			v, ok := val.([]byte)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := [12]byte{}
			copy(vv[:], v)
			row.GUDID = &vv
		case 3:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.Generation = &vv
		case 4:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.FirmwareVersion = &vv
		case 5:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.ProtocolVersion = &vv
		case 6:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			row.SpaceForIssuance = &vv
		case 7:
			vl, ok := val.(stream.List)
			if !ok {
				vl = stream.List{val}
			}
			for _, val := range vl {
				v, ok := val.([]byte)
				if !ok {
					return nil, core.ErrMalformedMethodResponse
				}
				row.SSC = append(row.SSC, string(v))
			}
		case 8:
			v, ok := val.(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			row.ProgrammaticResetEnable = &vv
		}
	}

	res[row.UID] = row
	return res, nil
}
