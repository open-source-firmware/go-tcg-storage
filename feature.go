// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// "Feature" encoding/decoding

package tcgstorage

import (
	"encoding/binary"
	"io"
)

type FeatureCode uint16

const (
	FeatureCodeTPer              FeatureCode = 0x0001
	FeatureCodeLocking           FeatureCode = 0x0002
	FeatureCodeGeometry          FeatureCode = 0x0003
	FeatureCodeSecureMsg         FeatureCode = 0x0004
	FeatureCodeEnterprise        FeatureCode = 0x0100
	FeatureCodeOpalV1            FeatureCode = 0x0200
	FeatureCodeSingleUser        FeatureCode = 0x0201
	FeatureCodeDataStore         FeatureCode = 0x0202
	FeatureCodeOpalV2            FeatureCode = 0x0203
	FeatureCodeOpalite           FeatureCode = 0x0301
	FeatureCodePyriteV1          FeatureCode = 0x0302
	FeatureCodePyriteV2          FeatureCode = 0x0303
	FeatureCodeRubyV1            FeatureCode = 0x0304
	FeatureCodeLockingLBA        FeatureCode = 0x0401
	FeatureCodeBlockSID          FeatureCode = 0x0402
	FeatureCodeNamespaceLocking  FeatureCode = 0x0403
	FeatureCodeDataRemoval       FeatureCode = 0x0404
	FeatureCodeNamespaceGeometry FeatureCode = 0x0405
)

type FeatureTPer struct {
	SyncSupported       bool
	AsyncSupported      bool
	AckNakSupported     bool
	BufferMgmtSupported bool
	StreamingSupported  bool
	ComIDMgmtSupported  bool
}

type FeatureLocking struct {
	LockingSupported bool
	LockingEnabled   bool
	Locked           bool
	MediaEncryption  bool
	MBREnabled       bool
	MBRDone          bool
}

type FeatureGeometry struct {
	// TODO
}
type FeatureSecureMsg struct {
	// TODO
}
type FeatureEnterprise struct {
	// TODO
}
type FeatureOpalV1 struct {
	// TODO
}
type FeatureSingleUser struct {
	// TODO
}
type FeatureDataStore struct {
	// TODO
}

type FeatureOpalV2 struct {
	BaseComID                     uint16
	NumComID                      uint16
	NumLockingSPAdminSupported    uint16
	NumLockingSPUserSupported     uint16
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

type FeatureOpalite struct {
	// TODO
}

type FeaturePyriteV1 struct {
	BaseComID                     uint16
	NumComID                      uint16
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

type FeaturePyriteV2 struct {
	BaseComID                     uint16
	NumComID                      uint16
	InitialCPINSIDIndicator       uint8
	BehaviorCPINSIDuponTPerRevert uint8
}

type FeatureRubyV1 struct {
	// TODO
}
type FeatureLockingLBA struct {
	// TODO
}
type FeatureBlockSID struct {
	// TODO
}
type FeatureNamespaceLocking struct {
	// TODO
}
type FeatureDataRemoval struct {
	// TODO
}
type FeatureNamespaceGeometry struct {
	// TODO
}

func readTPerFeature(rdr io.Reader) (*FeatureTPer, error) {
	f := &FeatureTPer{}
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

func readLockingFeature(rdr io.Reader) (*FeatureLocking, error) {
	f := &FeatureLocking{}
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
	return f, nil
}

func readGeometryFeature(rdr io.Reader) (*FeatureGeometry, error) {
	f := &FeatureGeometry{}
	return f, nil
}

func readSecureMsgFeature(rdr io.Reader) (*FeatureSecureMsg, error) {
	f := &FeatureSecureMsg{}
	return f, nil
}

func readEnterpriseFeature(rdr io.Reader) (*FeatureEnterprise, error) {
	f := &FeatureEnterprise{}
	return f, nil
}

func readOpalV1Feature(rdr io.Reader) (*FeatureOpalV1, error) {
	f := &FeatureOpalV1{}
	return f, nil
}

func readSingleUserFeature(rdr io.Reader) (*FeatureSingleUser, error) {
	f := &FeatureSingleUser{}
	return f, nil
}

func readDataStoreFeature(rdr io.Reader) (*FeatureDataStore, error) {
	f := &FeatureDataStore{}
	return f, nil
}

func readOpalV2Feature(rdr io.Reader) (*FeatureOpalV2, error) {
	f := &FeatureOpalV2{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func readOpaliteFeature(rdr io.Reader) (*FeatureOpalite, error) {
	f := &FeatureOpalite{}
	return f, nil
}

func readPyriteV1Feature(rdr io.Reader) (*FeaturePyriteV1, error) {
	f := &FeaturePyriteV1{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func readPyriteV2Feature(rdr io.Reader) (*FeaturePyriteV2, error) {
	f := &FeaturePyriteV2{}
	if err := binary.Read(rdr, binary.BigEndian, f); err != nil {
		return nil, err
	}
	return f, nil
}

func readRubyV1Feature(rdr io.Reader) (*FeatureRubyV1, error) {
	f := &FeatureRubyV1{}
	return f, nil
}

func readLockingLBAFeature(rdr io.Reader) (*FeatureLockingLBA, error) {
	f := &FeatureLockingLBA{}
	return f, nil
}

func readBlockSIDFeature(rdr io.Reader) (*FeatureBlockSID, error) {
	f := &FeatureBlockSID{}
	return f, nil
}

func readNamespaceLockingFeature(rdr io.Reader) (*FeatureNamespaceLocking, error) {
	f := &FeatureNamespaceLocking{}
	return f, nil
}

func readDataRemovalFeature(rdr io.Reader) (*FeatureDataRemoval, error) {
	f := &FeatureDataRemoval{}
	return f, nil
}

func readNamespaceGeometryFeature(rdr io.Reader) (*FeatureNamespaceGeometry, error) {
	f := &FeatureNamespaceGeometry{}
	return f, nil
}
