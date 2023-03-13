// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// "Feature" encoding/decoding

package feature

import (
	"encoding/binary"
	"io"
)

type FeatureCode uint16

const (
	CodeTPer              FeatureCode = 0x0001
	CodeLocking           FeatureCode = 0x0002
	CodeGeometry          FeatureCode = 0x0003
	CodeSecureMsg         FeatureCode = 0x0004
	CodeEnterprise        FeatureCode = 0x0100
	CodeOpalV1            FeatureCode = 0x0200
	CodeSingleUser        FeatureCode = 0x0201
	CodeDataStore         FeatureCode = 0x0202
	CodeOpalV2            FeatureCode = 0x0203
	CodeOpalite           FeatureCode = 0x0301
	CodePyriteV1          FeatureCode = 0x0302
	CodePyriteV2          FeatureCode = 0x0303
	CodeRubyV1            FeatureCode = 0x0304
	CodeLockingLBA        FeatureCode = 0x0401
	CodeBlockSID          FeatureCode = 0x0402
	CodeNamespaceLocking  FeatureCode = 0x0403
	CodeDataRemoval       FeatureCode = 0x0404
	CodeNamespaceGeometry FeatureCode = 0x0405
	CodeSeagatePorts      FeatureCode = 0xC001
)

type TPer struct {
	SyncSupported       bool
	AsyncSupported      bool
	AckNakSupported     bool
	BufferMgmtSupported bool
	StreamingSupported  bool
	ComIDMgmtSupported  bool
}

type Locking struct {
	LockingSupported bool
	LockingEnabled   bool
	Locked           bool
	MediaEncryption  bool
	MBREnabled       bool
	MBRDone          bool
	MBRShadowing     bool
}

type CommonSSC struct {
	BaseComID uint16
	NumComID  uint16
}

type Geometry struct {
	// TODO
}
type SecureMsg struct {
	// TODO
}

type Enterprise struct {
	CommonSSC
	RangeCrossingBehavior bool
}

type OpalV1 struct {
	// TODO
}
type SingleUser struct {
	// TODO
}
type DataStore struct {
	// TODO
}

type OpalV2 struct {
	CommonSSC
	RangeCrossingBehavior         bool
	NumLockingSPAdminSupported    uint16
	NumLockingSPUserSupported     uint16
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

type Opalite struct {
	// TODO
}

type PyriteV1 struct {
	CommonSSC
	_                             [4]byte
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

type PyriteV2 struct {
	CommonSSC
	_                             [4]byte
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

// 3.1.1.5 Ruby SSC V1.00 Feature (Feature Code = 0x0304)
type RubyV1 struct {
	CommonSSC
	RangeCrossingBehavior         bool
	NumLockingSPAdminSupported    uint16
	NumLockingSPUserSupported     uint16
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}
type LockingLBA struct {
	// TODO
}

type BlockSID struct {
	LockingSPFreezeLockState      bool
	LockingSPFreezeLockSupported  bool
	SIDAuthenticationBlockedState bool
	SIDValueState                 bool
	HardwareReset                 bool
}

type NamespaceLocking struct {
	// TODO
}
type DataRemoval struct {
	// TODO
}
type NamespaceGeometry struct {
	// TODO
}

type SeagatePort struct {
	PortIdentifier int32
	PortLocked     uint8
}

type SeagatePorts struct {
	Ports []SeagatePort
}

func ReadTPerFeature(rdr io.Reader) (*TPer, error) {
	f := &TPer{}
	var raw uint8
	if err := binary.Read(rdr, binary.BigEndian, &raw); err != nil {
		return nil, err
	}
	f.SyncSupported = raw&0x1 > 0
	f.AsyncSupported = raw&0x2 > 0
	f.AckNakSupported = raw&0x4 > 0
	f.BufferMgmtSupported = raw&0x8 > 0
	f.StreamingSupported = raw&0x10 > 0
	f.ComIDMgmtSupported = raw&0x40 > 0
	return f, nil
}

func ReadLockingFeature(rdr io.Reader) (*Locking, error) {
	f := &Locking{}
	var raw uint8
	if err := binary.Read(rdr, binary.BigEndian, &raw); err != nil {
		return nil, err
	}
	f.LockingSupported = raw&0x1 > 0
	f.LockingEnabled = raw&0x2 > 0
	f.Locked = raw&0x4 > 0
	f.MediaEncryption = raw&0x8 > 0
	f.MBREnabled = raw&0x10 > 0
	f.MBRDone = raw&0x20 > 0
	// If MBR Shadowing feature is absent (i.e., is not supported), then this bit SHALL be 1.
	f.MBRShadowing = raw&0x40 < 1
	return f, nil
}

func ReadGeometryFeature(rdr io.Reader) (*Geometry, error) {
	f := &Geometry{}
	return f, nil
}

func ReadSecureMsgFeature(rdr io.Reader) (*SecureMsg, error) {
	f := &SecureMsg{}
	return f, nil
}

func ReadEnterpriseFeature(rdr io.Reader) (*Enterprise, error) {
	f := &Enterprise{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func ReadOpalV1Feature(rdr io.Reader) (*OpalV1, error) {
	f := &OpalV1{}
	return f, nil
}

func ReadSingleUserFeature(rdr io.Reader) (*SingleUser, error) {
	f := &SingleUser{}
	return f, nil
}

func ReadDataStoreFeature(rdr io.Reader) (*DataStore, error) {
	f := &DataStore{}
	return f, nil
}

func ReadOpalV2Feature(rdr io.Reader) (*OpalV2, error) {
	f := &OpalV2{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func ReadOpaliteFeature(rdr io.Reader) (*Opalite, error) {
	f := &Opalite{}
	return f, nil
}

func ReadPyriteV1Feature(rdr io.Reader) (*PyriteV1, error) {
	f := &PyriteV1{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func ReadPyriteV2Feature(rdr io.Reader) (*PyriteV2, error) {
	f := &PyriteV2{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func ReadRubyV1Feature(rdr io.Reader) (*RubyV1, error) {
	f := &RubyV1{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func ReadLockingLBAFeature(rdr io.Reader) (*LockingLBA, error) {
	f := &LockingLBA{}
	return f, nil
}

func ReadBlockSIDFeature(rdr io.Reader) (*BlockSID, error) {
	f := &BlockSID{}
	var raw uint8
	if err := binary.Read(rdr, binary.BigEndian, &raw); err != nil {
		return nil, err
	}
	f.SIDValueState = raw&0x1 > 0
	f.SIDAuthenticationBlockedState = raw&0x2 > 0
	f.LockingSPFreezeLockSupported = raw&0x4 > 0
	f.LockingSPFreezeLockState = raw&0x8 > 0
	if err := binary.Read(rdr, binary.BigEndian, &raw); err != nil {
		return nil, err
	}
	f.HardwareReset = raw&0x1 > 0
	return f, nil
}

func ReadNamespaceLockingFeature(rdr io.Reader) (*NamespaceLocking, error) {
	f := &NamespaceLocking{}
	return f, nil
}

func ReadDataRemovalFeature(rdr io.Reader) (*DataRemoval, error) {
	f := &DataRemoval{}
	return f, nil
}

func ReadNamespaceGeometryFeature(rdr io.Reader) (*NamespaceGeometry, error) {
	f := &NamespaceGeometry{}
	return f, nil
}

func ReadSeagatePorts(rdr io.Reader) (*SeagatePorts, error) {
	f := &SeagatePorts{}
	for {
		p := SeagatePort{}
		d := struct {
			Ident int32
			State uint8
			_     [3]byte
		}{}
		if err := binary.Read(rdr, binary.BigEndian, &d); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		p.PortIdentifier = d.Ident
		p.PortLocked = d.State
		f.Ports = append(f.Ports, p)
	}
	return f, nil
}
