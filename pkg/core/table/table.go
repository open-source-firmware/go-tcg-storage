// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"errors"
	"fmt"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/stream"
)

type RowUID [8]byte
type TableUID [8]byte

func (t *TableUID) Row(uid [4]byte) RowUID {
	return [8]byte{t[0], t[1], t[2], t[3], uid[0], uid[1], uid[2], uid[3]}
}

var (
	CellBlock_StartRow    uint = 1
	CellBlock_EndRow      uint = 2
	CellBlock_StartColumn uint = 3
	CellBlock_EndColumn   uint = 4

	Table_ColumnUID uint = 0

	MethodIDEnterpriseGet          core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x06}
	MethodIDEnterpriseSet          core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x07}
	MethodIDGetACL                 core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x0D}
	MethodIDGet                    core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x16}
	MethodIDSet                    core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x17}
	MethodIDNext                   core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x08}
	MethodIDAuthenticate           core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x1C}
	MethodIDEnterpriseAuthenticate core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x0C}
	MethodIDRandom                 core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x06, 0x01}

	ErrEmptyResult = errors.New("empty result")
)

func GetCell(s *core.Session, row RowUID, column uint, columnName string) (interface{}, error) {
	m, err := GetPartialRow(s, row, column, columnName, column, columnName)
	if err != nil {
		return nil, err
	}
	for _, v := range m {
		return v, nil
	}
	return nil, ErrEmptyResult
}

func GetPartialRow(s *core.Session, row RowUID, startCol uint, startColName string, endCol uint, endColName string) (map[string]interface{}, error) {
	getUID := core.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(getUID[:], MethodIDEnterpriseGet[:])
	} else {
		copy(getUID[:], MethodIDGet[:])
	}
	mc := s.NewMethodCall(core.InvokingID(row), getUID)
	mc.StartList()
	mc.StartOptionalParameter(CellBlock_StartColumn, "startColumn")
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		mc.Bytes([]byte(startColName))
	} else {
		mc.UInt(startCol)
	}
	mc.EndOptionalParameter()
	mc.StartOptionalParameter(CellBlock_EndColumn, "endColumn")
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		mc.Bytes([]byte(endColName))
	} else {
		mc.UInt(endCol)
	}
	mc.EndOptionalParameter()
	mc.EndList()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	// The Enterprise Get has an extra level of lists
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		var ok bool
		resp, ok = resp[0].(stream.List)
		if !ok {
			return nil, core.ErrMalformedMethodResponse
		}
	}
	val, err := parseGetResult(resp)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrEmptyResult
	}
	return val, nil
}

func GetFullRow(s *core.Session, row RowUID) (map[string]interface{}, error) {
	getUID := core.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(getUID[:], MethodIDEnterpriseGet[:])
	} else {
		copy(getUID[:], MethodIDGet[:])
	}
	mc := s.NewMethodCall(core.InvokingID(row), getUID)
	mc.StartList()
	mc.EndList()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	// The Enterprise Get has an extra level of lists
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		var ok bool
		resp, ok = resp[0].(stream.List)
		if !ok {
			return nil, core.ErrMalformedMethodResponse
		}
	}
	val, err := parseGetResult(resp)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, ErrEmptyResult
	}
	return val, nil
}

func Enumerate(s *core.Session, table TableUID) ([]RowUID, error) {
	mc := s.NewMethodCall(core.InvokingID(table), MethodIDNext)
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	result, ok := resp[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	uidrefs, ok := result[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	res := []RowUID{}
	for _, ur := range uidrefs {
		br, ok := ur.([]byte)
		if !ok || len(br) != 8 {
			return nil, core.ErrMalformedMethodResponse
		}
		r := RowUID{}
		copy(r[:], br)
		res = append(res, r)
	}
	return res, nil
}

func parseGetResult(res stream.List) (map[string]interface{}, error) {
	methodResult, ok := res[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	if len(methodResult) == 0 {
		return nil, ErrEmptyResult
	}
	inner, ok := methodResult[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	if len(inner) == 0 {
		return nil, ErrEmptyResult
	}
	return parseRowValues(inner)
}

// Parse a RowValues return value into a map.
//
// Due to the Enterprise SSC relying on sending ASCII column names instead of
// uinteger IDs as the Core V2.0 spec does, we have to support both.
func parseRowValues(rv stream.List) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	for i := range rv {
		if stream.EqualToken(rv[i], stream.StartName) {
			colID, okID := rv[i+1].(uint)
			colRawName, okString := rv[i+1].([]byte)
			if !okID && !okString {
				return nil, core.ErrMalformedMethodResponse
			}
			colName := ""
			if okID {
				colName = fmt.Sprintf("%d", colID)
			}
			if okString {
				colName = string(colRawName)
			}
			if !stream.EqualToken(rv[i+2], stream.EndName) {
				res[colName] = rv[i+2]
			}
		}
	}
	return res, nil
}

func NewSetCall(s *core.Session, row RowUID) *core.MethodCall {
	setUID := core.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(setUID[:], MethodIDEnterpriseSet[:])
	} else {
		copy(setUID[:], MethodIDSet[:])
	}
	mc := s.NewMethodCall(core.InvokingID(row), setUID)
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		// The two first arguments in ESET are required, and RowValues has an extra list
		mc.StartList()
		mc.EndList()
		mc.StartList()
		mc.StartList()
	} else {
		mc.StartOptionalParameter(1, "Values")
		mc.StartList()
	}
	return mc
}

func FinishSetCall(s *core.Session, mc *core.MethodCall) {
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		mc.EndList()
		mc.EndList()
	} else {
		mc.EndList()
		mc.EndOptionalParameter()
	}
}
