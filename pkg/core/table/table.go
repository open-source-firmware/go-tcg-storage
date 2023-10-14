// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"errors"
	"fmt"

	"github.com/matfax/go-tcg-storage/pkg/core"
	"github.com/matfax/go-tcg-storage/pkg/core/method"
	"github.com/matfax/go-tcg-storage/pkg/core/stream"
	"github.com/matfax/go-tcg-storage/pkg/core/uid"
)

type TableUID [8]byte

var (
	CellBlock_StartRow    uint = 1
	CellBlock_EndRow      uint = 2
	CellBlock_StartColumn uint = 3
	CellBlock_EndColumn   uint = 4

	Table_ColumnUID uint = 0

	ErrEmptyResult = errors.New("empty result")
)

func GetCell(s *core.Session, row uid.RowUID, column uint, columnName string) (interface{}, error) {
	m, err := GetPartialRow(s, row, column, columnName, column, columnName)
	if err != nil {
		return nil, err
	}
	for _, v := range m {
		return v, nil
	}
	return nil, ErrEmptyResult
}

func GetPartialRow(s *core.Session, row uid.RowUID, startCol uint, startColName string, endCol uint, endColName string) (map[string]interface{}, error) {
	getUID := uid.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(getUID[:], uid.OpalEnterpriseGet[:])
	} else {
		copy(getUID[:], uid.OpalGet[:])
	}
	mc := method.NewMethodCall(uid.InvokingID(row), getUID, s.MethodFlags)
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
			return nil, method.ErrMalformedMethodResponse
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

func GetFullRow(s *core.Session, row uid.RowUID) (map[string]interface{}, error) {
	getUID := uid.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(getUID[:], uid.OpalEnterpriseGet[:])
	} else {
		copy(getUID[:], uid.OpalGet[:])
	}
	mc := method.NewMethodCall(uid.InvokingID(row), getUID, s.MethodFlags)
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
			return nil, method.ErrMalformedMethodResponse
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

func Enumerate(s *core.Session, table uid.TableUID) ([]uid.RowUID, error) {
	mc := method.NewMethodCall(uid.InvokingID(table), uid.OpalNext, s.MethodFlags)
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	result, ok := resp[0].(stream.List)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	uidrefs, ok := result[0].(stream.List)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	res := []uid.RowUID{}
	for _, ur := range uidrefs {
		br, ok := ur.([]byte)
		if !ok || len(br) != 8 {
			return nil, method.ErrMalformedMethodResponse
		}
		r := uid.RowUID{}
		copy(r[:], br)
		res = append(res, r)
	}
	return res, nil
}

func parseGetResult(res stream.List) (map[string]interface{}, error) {
	methodResult, ok := res[0].(stream.List)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	if len(methodResult) == 0 {
		return nil, ErrEmptyResult
	}
	inner, ok := methodResult[0].(stream.List)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
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
				return nil, method.ErrMalformedMethodResponse
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

func NewSetCall(s *core.Session, row uid.RowUID) *method.MethodCall {
	setUID := uid.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(setUID[:], uid.OpalEnterpriseSet[:])
	} else {
		copy(setUID[:], uid.OpalSet[:])
	}
	mc := method.NewMethodCall(uid.InvokingID(row), setUID, s.MethodFlags)
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

func FinishSetCall(s *core.Session, mc *method.MethodCall) {
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		mc.EndList()
		mc.EndList()
	} else {
		mc.EndList()
		mc.EndOptionalParameter()
	}
}
