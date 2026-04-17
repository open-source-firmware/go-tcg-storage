// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations on Locking SP tables

package table

import (
	"errors"
	"fmt"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/method"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/stream"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/uid"
)

var ErrMBRNotSupproted = errors.New("drive does not support MBR")

type (
	EncryptSupport     uint
	KeysAvailableConds uint
)

type ResetType uint

const (
	ResetPowerOff ResetType = 0
	ResetHardware ResetType = 1
	ResetHotPlug  ResetType = 2
	// The parameter number for KeepGlobalRangeKey SHALL be 0x060000
	// TCG Storage Security Subsystem Class: Opal | Version 2.02 | Revision 1.0 | Page 86
	KeepGlobalRangeKey uint = 0x060000
)

type ProtectMechanism uint

const (
	VendorUnique               ProtectMechanism = 0
	AuthenticationDataRequired ProtectMechanism = 1
)

type SecretProtect struct {
	UID              uid.UID
	Table            uid.RowUID
	Column           uint
	ProtectMechanism []ProtectMechanism
}

const ProtectMechanismColumn uint = 3

type LockingInfoRow struct {
	UID                  uid.RowUID
	Name                 *string
	Version              *uint32
	EncryptSupport       *EncryptSupport
	MaxRanges            *uint32
	MaxReEncryptions     *uint32
	KeysAvailableCfg     *KeysAvailableConds
	AlignmentRequired    *bool
	LogicalBlockSize     *uint32
	AlignmentGranularity *uint64
	LowestAlignedLBA     *uint64
}

func LockingSPActivate(s *core.Session) error {
	var lockingsp uid.InvokingID
	copy(lockingsp[:], uid.LockingSP[:])
	mc := method.NewMethodCall(lockingsp, uid.MethodIDActivate, s.MethodFlags)
	_, err := s.ExecuteMethod(mc)
	if err != nil {
		return err
	}
	return nil
}

func LockingSecretProtect(s *core.Session) ([]SecretProtect, error) {
	if uids, err := Enumerate(s, uid.Locking_SecretProtect); err != nil {
		return nil, err
	} else {
		result := make([]SecretProtect, len(uids))
		for i, rowUid := range uids {
			val, err := GetFullRow(s, rowUid)
			if err != nil {
				return nil, err
			}

			for col, val := range val {
				switch col {
				case "0", "UID":
					v, ok := val.([]byte)
					if !ok {
						return nil, method.ErrMalformedMethodResponse
					}
					copy(result[i].UID[:], v[:8])
				case "1", "Table":
					v, ok := val.([]byte)
					if !ok {
						return nil, method.ErrMalformedMethodResponse
					}
					copy(result[i].Table[:], v[:8])
				case "2", "Column":
					v, ok := val.(uint)
					if !ok {
						return nil, method.ErrMalformedMethodResponse
					}
					result[i].Column = v
				case "3", "ProtectMechanisms":
					v, ok := val.(stream.List)
					if !ok {
						return nil, method.ErrMalformedMethodResponse
					}
					mechanisms := make([]ProtectMechanism, len(v))
					for n, val := range v {
						mechanism, ok := val.(uint)
						if !ok {
							return nil, method.ErrMalformedMethodResponse
						}
						mechanisms[n] = ProtectMechanism(mechanism)
					}
					result[i].ProtectMechanism = mechanisms
				}
			}
		}
		return result, nil
	}
}

func LockingInfo(s *core.Session) (*LockingInfoRow, error) {
	rowUID := uid.RowUID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(rowUID[:], uid.EnterpriseLockingInfoObj[:])
	} else {
		copy(rowUID[:], uid.LockingInfoObj[:])
	}

	val, err := GetFullRow(s, rowUID)
	if err != nil {
		return nil, err
	}
	row := LockingInfoRow{}
	for col, val := range val {
		switch col {
		case "0", "UID":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			copy(row.UID[:], v[:8])
		case "1", "Name":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := string(v)
			row.Name = &vv
		case "2", "Version":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.Version = &vv
		case "3", "EncryptSupport":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := EncryptSupport(v)
			row.EncryptSupport = &vv
		case "4", "MaxRanges":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.MaxRanges = &vv
		case "5", "MaxReEncryptions":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.MaxReEncryptions = &vv
		case "6", "KeysAvailableCfg":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := KeysAvailableConds(v)
			row.KeysAvailableCfg = &vv
		case "7":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			row.AlignmentRequired = &vv
		case "8":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.LogicalBlockSize = &vv
		case "9":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			row.AlignmentGranularity = &vv
		case "10":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			row.LowestAlignedLBA = &vv
		}
	}
	return &row, nil
}

func Locking_Enumerate(s *core.Session) ([]uid.RowUID, error) {
	return Enumerate(s, uid.Locking_LockingTable)
}

type LockingRow struct {
	UID              uid.RowUID
	Name             *string
	RangeStart       *uint64
	RangeLength      *uint64
	ReadLockEnabled  *bool
	WriteLockEnabled *bool
	ReadLocked       *bool
	WriteLocked      *bool
	LockOnReset      []ResetType
	ActiveKey        *uid.RowUID
	// NOTE: There are more fields in the standards that have been omited
}

func Locking_Get(s *core.Session, row uid.RowUID) (*LockingRow, error) {
	val, err := GetFullRow(s, row)
	if err != nil {
		return nil, err
	}
	lr := LockingRow{}
	for col, val := range val {
		switch col {
		case "0", "UID":
			v, ok := val.([]byte)
			if !ok || len(v) != 8 {
				return nil, method.ErrMalformedMethodResponse
			}
			copy(lr.UID[:], v[:8])
		case "1", "Name":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := string(v)
			lr.Name = &vv
		case "3", "RangeStart":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			lr.RangeStart = &vv
		case "4", "RangeLength":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint64(v)
			lr.RangeLength = &vv
		case "5", "ReadLockEnabled":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			lr.ReadLockEnabled = &vv
		case "6", "WriteLockEnabled":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			lr.WriteLockEnabled = &vv
		case "7", "ReadLocked":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			lr.ReadLocked = &vv
		case "8", "WriteLocked":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			lr.WriteLocked = &vv
		case "9", "LockOnReset":
			vl, ok := val.(stream.List)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			for _, val := range vl {
				v, ok := val.(uint)
				if !ok {
					return nil, method.ErrMalformedMethodResponse
				}
				lr.LockOnReset = append(lr.LockOnReset, ResetType(v))
			}
		case "10", "ActiveKey":
			v, ok := val.([]byte)
			if !ok || len(v) != 8 {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uid.RowUID{}
			copy(vv[:], v)
			lr.ActiveKey = &vv
		}
	}
	return &lr, nil
}

func ConfigureLockingRange(s *core.Session) error {
	var row [8]byte
	copy(row[:], uid.LockingGlobalRange[:])
	mc := NewSetCall(s, row)
	mc.Token(stream.StartName)
	mc.Token(stream.ReadLockEnabled)
	mc.Token(stream.OpalFalse)
	mc.Token(stream.EndName)
	mc.Token(stream.StartName)
	mc.Token(stream.WriteLockEnabled)
	mc.Token(stream.OpalFalse)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndOptionalParameter()
	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}
	return nil
}

func Locking_Set(s *core.Session, row *LockingRow) error {
	mc := NewSetCall(s, row.UID)

	if row.Name != nil {
		mc.StartOptionalParameter(1, "Name")
		mc.Bytes([]byte(*row.Name))
		mc.EndOptionalParameter()
	}

	if row.RangeStart != nil {
		mc.StartOptionalParameter(3, "RangeStart")
		mc.UInt(uint(*row.RangeStart))
		mc.EndOptionalParameter()
	}

	if row.RangeLength != nil {
		mc.StartOptionalParameter(4, "RangeLength")
		mc.UInt(uint(*row.RangeLength))
		mc.EndOptionalParameter()
	}

	if row.ReadLockEnabled != nil {
		mc.StartOptionalParameter(5, "ReadLockEnabled")
		mc.Bool(*row.ReadLockEnabled)
		mc.EndOptionalParameter()
	}
	if row.WriteLockEnabled != nil {
		mc.StartOptionalParameter(6, "WriteLockEnabled")
		mc.Bool(*row.WriteLockEnabled)
		mc.EndOptionalParameter()
	}
	if row.ReadLocked != nil {
		mc.StartOptionalParameter(7, "ReadLocked")
		mc.Bool(*row.ReadLocked)
		mc.EndOptionalParameter()
	}

	if row.WriteLocked != nil {
		mc.StartOptionalParameter(8, "WriteLocked")
		mc.Bool(*row.WriteLocked)
		mc.EndOptionalParameter()
	}

	// TODO: Add these columns
	// mc.StartOptionalParameter(9, "LockOnReset")
	// mc.StartOptionalParameter(10, "ActiveKey")

	FinishSetCall(s, mc)
	_, err := s.ExecuteMethod(mc)
	return err
}

// Admin_C_Pin_Admin1_SetPIN sets the SID Pin in the Admin_C_PIN_Table
func Admin_C_Pin_Admin1_SetPIN(s *core.Session, password []byte) error {
	// password needs to be hashed somehow.
	if len(password) < 16 {
		return fmt.Errorf("invalid length of password hash")
	}
	mc := NewSetCall(s, uid.Admin_C_PIN_Admin1Row)
	mc.Token(stream.StartName)
	mc.Token(stream.OpalPIN)
	mc.Bytes(password)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndOptionalParameter()

	_, err := s.ExecuteMethod(mc)
	if err != nil {
		return err
	}
	return nil
}

type MBRControl struct {
	Enable         *bool
	Done           *bool
	MBRDoneOnReset *[]ResetType
}

func MBRControl_Set(s *core.Session, row *MBRControl) error {
	mc := NewSetCall(s, uid.MBRControlObj)

	if row.Enable != nil {
		mc.StartOptionalParameter(1, "Enable")
		mc.Bool(*row.Enable)
		mc.EndOptionalParameter()
	}
	if row.Done != nil {
		mc.StartOptionalParameter(2, "Done")
		mc.Bool(*row.Done)
		mc.EndOptionalParameter()
	}
	if row.MBRDoneOnReset != nil {
		mc.StartOptionalParameter(3, "MBRDoneOnReset")
		mc.StartList()
		for _, x := range *row.MBRDoneOnReset {
			mc.UInt(uint(x))
		}
		mc.EndList()
		mc.EndOptionalParameter()
	}
	FinishSetCall(s, mc)
	_, err := s.ExecuteMethod(mc)
	return err
}

type MBRTableInfo struct {
	// Size in bytes
	Size uint32

	// If set, writes need to be a multiple of this value
	MandatoryWriteGranularity uint32

	// If set, reads are recommended to be aligned to this value
	RecommendedAccessGranularity uint32
}

// SuggestBufferSize returns a safe maximum payload length for a single
// MBR_Read or MBR_Write call.
//
// Although reads (TPer → Host) are formally bounded by HostProperties and
// writes (Host → TPer) by TPerProperties, many drives seem to have firmware
// that has troubles with transfers larger than what they themselves can
// receive — we lean to safe smaller bounds.
//
// The result is aligned down to MandatoryWriteGranularity and
// RecommendedAccessGranularity so it is also a legal write length.
func (m *MBRTableInfo) SuggestBufferSize(s *core.Session) uint {
	ms := s.ControlSession.TPerProperties.MaxIndTokenSize
	if s.ControlSession.TPerProperties.MaxAggTokenSize > ms {
		ms = s.ControlSession.TPerProperties.MaxAggTokenSize
	}
	// Reserve room for method framing (Where/Values names, long-atom header,
	// status list). 128 B matches what sedutil-cli leaves free and is
	// comfortably larger than the actual overhead in either direction.
	ms -= 128
	ms = ms & ^uint(m.MandatoryWriteGranularity-1)
	ms = ms & ^uint(m.RecommendedAccessGranularity-1)
	return ms
}

func MBR_TableInfo(s *core.Session) (*MBRTableInfo, error) {
	tcol, err := GetFullRow(s, uid.Base_TableRowForTable(uid.Locking_MBRTable))
	if err != nil {
		if err == ErrEmptyResult {
			return nil, ErrMBRNotSupproted
		}
		return nil, err
	}

	mi := &MBRTableInfo{
		MandatoryWriteGranularity:    1,
		RecommendedAccessGranularity: 1,
	}
	// Enterprise does not support MBR so don't bother setting the text columns
	for col, val := range tcol {
		switch col {
		case "7":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			mi.Size = uint32(v)
		case "13":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			mi.MandatoryWriteGranularity = uint32(v)
		case "14":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			mi.RecommendedAccessGranularity = uint32(v)
		}
	}

	if mi.Size == 0 {
		return nil, errors.New("device did not specify MBR size")
	}
	return mi, nil
}

func MBR_Read(s *core.Session, p []byte, off uint32) (int, error) {
	mc := method.NewMethodCall(uid.InvokingID(uid.Locking_MBRTable), uid.OpalGet, s.MethodFlags)
	mc.StartList()
	mc.StartOptionalParameter(CellBlock_StartRow, "startRow")
	mc.UInt(uint(off))
	mc.EndOptionalParameter()
	mc.StartOptionalParameter(CellBlock_EndRow, "endRow")
	mc.UInt(uint(off) + uint(len(p)) - 1)
	mc.EndOptionalParameter()
	mc.EndList()
	res, err := s.ExecuteMethod(mc)
	if err != nil {
		return 0, err
	}
	methodResult, ok := res[0].(stream.List)
	if !ok {
		return 0, method.ErrMalformedMethodResponse
	}
	if len(methodResult) == 0 {
		return 0, ErrEmptyResult
	}
	inner, ok := methodResult[0].([]uint8)
	if !ok {
		return 0, method.ErrMalformedMethodResponse
	}
	if len(inner) == 0 {
		return 0, ErrEmptyResult
	}

	l := len(inner)
	if len(p) < l {
		l = len(p)
	}
	copy(p, inner[:l])
	return l, nil
}

// MBR_Write writes len(p) bytes from p to the shadow MBR byte table starting
// at byte offset off. It performs a single Set method call, so len(p) must fit
// in one token — use MBRTableInfo.SuggestWriteBufferSize to pick a safe chunk
// size and let the caller loop. The caller is also responsible for respecting
// MBRTableInfo.MandatoryWriteGranularity (both off and len(p) must be
// multiples of it) and for staying within MBRTableInfo.Size.
//
// On success returns len(p), nil.
func MBR_Write(s *core.Session, p []byte, off uint32) (int, error) {
	mc := method.NewMethodCall(uid.InvokingID(uid.Locking_MBRTable), uid.OpalSet, s.MethodFlags)
	mc.StartOptionalParameter(uint(stream.OpalWhere), "Where")
	mc.UInt(uint(off))
	mc.EndOptionalParameter()
	mc.StartOptionalParameter(uint(stream.OpalValue), "Values")
	mc.Bytes(p)
	mc.EndOptionalParameter()
	if _, err := s.ExecuteMethod(mc); err != nil {
		return 0, err
	}
	return len(p), nil
}

func LoadPBAImage(s *core.Session, image []byte) error {
	mi, err := MBR_TableInfo(s)
	if err != nil {
		return fmt.Errorf("MBR_TableInfo failed: %v", err)
	}
	chunk := mi.SuggestBufferSize(s)
	if chunk == 0 {
		return errors.New("SuggestBufferSize returned 0")
	}
	for off := uint(0); off < uint(len(image)); off += chunk {
		end := off + chunk
		if end > uint(len(image)) {
			end = uint(len(image))
		}
		if _, err := MBR_Write(s, image[off:end], uint32(off)); err != nil {
			return fmt.Errorf("MBR_Write at offset %d failed: %v", off, err)
		}
	}
	return nil
}

func RevertLockingSP(s *core.Session, keep bool) error {
	mc := method.NewMethodCall(uid.InvokeIDThisSP, uid.OpalRevertSP, s.MethodFlags)
	if keep {
		mc.Token(stream.StartName)
		mc.UInt(KeepGlobalRangeKey)
		mc.Token(stream.OpalTrue)
		mc.Token(stream.EndName)
	}
	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}
	return nil
}

func SetBandMaster0Pin(s *core.Session, band0PinHash []byte) error {
	if s.ProtocolLevel != core.ProtocolLevelEnterprise {
		return fmt.Errorf("invalid Protocol Level for operation")
	}
	mc := NewSetCall(s, uid.Admin_C_Pin_BandMaster0)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("PIN"))
	mc.Bytes(band0PinHash)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndList()

	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}

	return nil
}

func SetEraseMasterPin(s *core.Session, erasePinHash []byte) error {
	if s.ProtocolLevel != core.ProtocolLevelEnterprise {
		return fmt.Errorf("invalid Protocol Level for operation")
	}
	mc := NewSetCall(s, uid.Admin_C_Pin_EraseMaster)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("PIN"))
	mc.Bytes(erasePinHash)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndList()

	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}

	return nil
}

func EraseBand(s *core.Session, band uid.InvokingID) error {
	if s.ProtocolLevel != core.ProtocolLevelEnterprise {
		return fmt.Errorf("invalid Protocol Level for operation")
	}

	mc := method.NewMethodCall(band, uid.MethodIDEraseEnterprise, s.MethodFlags)

	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}
	return nil
}

func EnableGlobalRangeEnterprise(s *core.Session) error {
	mc := NewSetCall(s, uid.GlobalRangeRowUID)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("ReadLockEnabled"))
	mc.Token(stream.OpalTrue)
	mc.Token(stream.EndName)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("WriteLockEnabled"))
	mc.Token(stream.OpalTrue)
	mc.Token(stream.EndName)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("ReadLocked"))
	mc.Token(stream.OpalTrue)
	mc.Token(stream.EndName)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("WriteLocked"))
	mc.Token(stream.OpalTrue)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndList()

	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}
	return nil
}

func UnlockGlobalRangeEnterprise(s *core.Session, band uid.RowUID) error {
	mc := NewSetCall(s, band)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("ReadLocked"))
	mc.Token(stream.OpalFalse)
	mc.Token(stream.EndName)
	mc.Token(stream.StartName)
	mc.Bytes([]byte("WriteLocked"))
	mc.Token(stream.OpalFalse)
	mc.Token(stream.EndName)
	mc.EndList()
	mc.EndList()

	if _, err := s.ExecuteMethod(mc); err != nil {
		return err
	}
	return nil
}
