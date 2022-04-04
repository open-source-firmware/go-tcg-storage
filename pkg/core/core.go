// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core/feature"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

type DriveIntf interface {
	IFRecv(proto drive.SecurityProtocol, sps uint16, data *[]byte) error
	IFSend(proto drive.SecurityProtocol, sps uint16, data []byte) error
}

type ComID int
type ComIDRequest [4]byte

const (
	ComIDInvalid     ComID = -1
	ComIDDiscoveryL0 ComID = 1
)

var (
	ComIDRequestVerifyComIDValid ComIDRequest = [4]byte{0x00, 0x00, 0x00, 0x01}
	ComIDRequestStackReset       ComIDRequest = [4]byte{0x00, 0x00, 0x00, 0x02}

	ErrNotSupported = errors.New("device does not support TCG Storage Core")
)

type Level0Discovery struct {
	MajorVersion      int
	MinorVersion      int
	Vendor            [32]byte
	TPer              *feature.TPer
	Locking           *feature.Locking
	Geometry          *feature.Geometry
	SecureMsg         *feature.SecureMsg
	Enterprise        *feature.Enterprise
	OpalV1            *feature.OpalV1
	SingleUser        *feature.SingleUser
	DataStore         *feature.DataStore
	OpalV2            *feature.OpalV2
	Opalite           *feature.Opalite
	PyriteV1          *feature.PyriteV1
	PyriteV2          *feature.PyriteV2
	RubyV1            *feature.RubyV1
	LockingLBA        *feature.LockingLBA
	BlockSID          *feature.BlockSID
	NamespaceLocking  *feature.NamespaceLocking
	DataRemoval       *feature.DataRemoval
	NamespaceGeometry *feature.NamespaceGeometry
	SeagatePorts      *feature.SeagatePorts
	UnknownFeatures   []uint16
}

// Request an (extended) ComID.
func GetComID(d DriveIntf) (ComID, error) {
	var comID [512]byte
	comIDs := comID[:]
	if err := d.IFRecv(drive.SecurityProtocolTCGTPer, 0, &comIDs); err != nil {
		return ComIDInvalid, err
	}

	c := binary.BigEndian.Uint16(comID[0:2])
	ce := binary.BigEndian.Uint16(comID[2:4])

	return ComID(uint32(c) + uint32(ce)<<16), nil
}

func HandleComIDRequest(d DriveIntf, comID ComID, req ComIDRequest) ([]byte, error) {
	var buf [512]byte
	binary.BigEndian.PutUint16(buf[0:2], uint16(comID&0xffff))
	binary.BigEndian.PutUint16(buf[2:4], uint16((comID&0xffff0000)>>16))
	copy(buf[4:8], req[:])

	if err := d.IFSend(drive.SecurityProtocolTCGTPer, uint16(comID&0xffff), buf[:]); err != nil {
		return nil, err
	}

	buf = [512]byte{}
	bufs := buf[:]
	if err := d.IFRecv(drive.SecurityProtocolTCGTPer, uint16(comID&0xffff), &bufs); err != nil {
		return nil, err
	}

	// TODO: Verify the request code in response?
	size := binary.BigEndian.Uint16(buf[10:12])
	return buf[12 : 12+size], nil
}

// Validate a ComID.
func IsComIDValid(d DriveIntf, comID ComID) (bool, error) {
	res, err := HandleComIDRequest(d, comID, ComIDRequestVerifyComIDValid)
	if err != nil {
		return false, err
	}
	state := binary.BigEndian.Uint32(res[0:4])
	return state == 2 || state == 3, nil
}

// Reset the state of the synchronous protocol stack.
func StackReset(d DriveIntf, comID ComID) error {
	res, err := HandleComIDRequest(d, comID, ComIDRequestStackReset)
	if err != nil {
		return err
	}
	if len(res) < 4 {
		// TODO: Implement stack reset pending re-poll
		return fmt.Errorf("stack reset is probably Pending, which is not supported")
	}
	success := binary.BigEndian.Uint32(res[0:4])
	if success != 0 {
		return fmt.Errorf("stack reset reported failure")
	}
	return nil
}

// Perform a Level 0 SSC Discovery.
func Discovery0(d DriveIntf) (*Level0Discovery, error) {
	d0raw := make([]byte, 2048)
	if err := d.IFRecv(drive.SecurityProtocolTCGManagement, uint16(ComIDDiscoveryL0), &d0raw); err != nil {
		if err == drive.ErrNotSupported {
			return nil, ErrNotSupported
		}
		return nil, err
	}
	d0 := &Level0Discovery{}
	d0buf := bytes.NewBuffer(d0raw)
	d0hdr := struct {
		Size   uint32
		Major  uint16
		Minor  uint16
		_      [8]byte
		Vendor [32]byte
	}{}
	if err := binary.Read(d0buf, binary.BigEndian, &d0hdr); err != nil {
		return nil, fmt.Errorf("failed to parse Level 0 discovery: %v", err)
	}
	if d0hdr.Size == 0 {
		return nil, ErrNotSupported
	}
	d0.MajorVersion = int(d0hdr.Major)
	d0.MinorVersion = int(d0hdr.Minor)
	copy(d0.Vendor[:], d0hdr.Vendor[:])

	fsize := int(d0hdr.Size) - binary.Size(d0hdr) + 4
	for fsize > 0 {
		fhdr := struct {
			Code    feature.FeatureCode
			Version uint8
			Size    uint8
		}{}
		if err := binary.Read(d0buf, binary.BigEndian, &fhdr); err != nil {
			return nil, fmt.Errorf("failed to parse feature header: %v", err)
		}
		frdr := io.LimitReader(d0buf, int64(fhdr.Size))
		var err error
		switch fhdr.Code {
		case feature.CodeTPer:
			d0.TPer, err = feature.ReadTPerFeature(frdr)
		case feature.CodeLocking:
			d0.Locking, err = feature.ReadLockingFeature(frdr)
		case feature.CodeGeometry:
			d0.Geometry, err = feature.ReadGeometryFeature(frdr)
		case feature.CodeSecureMsg:
			d0.SecureMsg, err = feature.ReadSecureMsgFeature(frdr)
		case feature.CodeEnterprise:
			d0.Enterprise, err = feature.ReadEnterpriseFeature(frdr)
		case feature.CodeOpalV1:
			d0.OpalV1, err = feature.ReadOpalV1Feature(frdr)
		case feature.CodeSingleUser:
			d0.SingleUser, err = feature.ReadSingleUserFeature(frdr)
		case feature.CodeDataStore:
			d0.DataStore, err = feature.ReadDataStoreFeature(frdr)
		case feature.CodeOpalV2:
			d0.OpalV2, err = feature.ReadOpalV2Feature(frdr)
		case feature.CodeOpalite:
			d0.Opalite, err = feature.ReadOpaliteFeature(frdr)
		case feature.CodePyriteV1:
			d0.PyriteV1, err = feature.ReadPyriteV1Feature(frdr)
		case feature.CodePyriteV2:
			d0.PyriteV2, err = feature.ReadPyriteV2Feature(frdr)
		case feature.CodeRubyV1:
			d0.RubyV1, err = feature.ReadRubyV1Feature(frdr)
		case feature.CodeLockingLBA:
			d0.LockingLBA, err = feature.ReadLockingLBAFeature(frdr)
		case feature.CodeBlockSID:
			d0.BlockSID, err = feature.ReadBlockSIDFeature(frdr)
		case feature.CodeNamespaceLocking:
			d0.NamespaceLocking, err = feature.ReadNamespaceLockingFeature(frdr)
		case feature.CodeDataRemoval:
			d0.DataRemoval, err = feature.ReadDataRemovalFeature(frdr)
		case feature.CodeNamespaceGeometry:
			d0.NamespaceGeometry, err = feature.ReadNamespaceGeometryFeature(frdr)
		case feature.CodeSeagatePorts:
			d0.SeagatePorts, err = feature.ReadSeagatePorts(frdr)
		default:
			// Unsupported feature
			d0.UnknownFeatures = append(d0.UnknownFeatures, uint16(fhdr.Code))
		}
		if err != nil {
			return nil, err
		}
		io.CopyN(ioutil.Discard, frdr, int64(fhdr.Size))
		fsize -= binary.Size(fhdr) + int(fhdr.Size)
	}
	return d0, nil
}
