// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package tcgstorage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bluecmd/go-tcg-storage/drive"
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
	TPer              *FeatureTPer
	Locking           *FeatureLocking
	Geometry          *FeatureGeometry
	SecureMsg         *FeatureSecureMsg
	Enterprise        *FeatureEnterprise
	OpalV1            *FeatureOpalV1
	SingleUser        *FeatureSingleUser
	DataStore         *FeatureDataStore
	OpalV2            *FeatureOpalV2
	Opalite           *FeatureOpalite
	PyriteV1          *FeaturePyriteV1
	PyriteV2          *FeaturePyriteV2
	RubyV1            *FeatureRubyV1
	LockingLBA        *FeatureLockingLBA
	BlockSID          *FeatureBlockSID
	NamespaceLocking  *FeatureNamespaceLocking
	DataRemoval       *FeatureDataRemoval
	NamespaceGeometry *FeatureNamespaceGeometry
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
			Code    FeatureCode
			Version uint8
			Size    uint8
		}{}
		if err := binary.Read(d0buf, binary.BigEndian, &fhdr); err != nil {
			return nil, fmt.Errorf("failed to parse feature header: %v", err)
		}
		frdr := io.LimitReader(d0buf, int64(fhdr.Size))
		var err error
		switch fhdr.Code {
		case FeatureCodeTPer:
			d0.TPer, err = readTPerFeature(frdr)
		case FeatureCodeLocking:
			d0.Locking, err = readLockingFeature(frdr)
		case FeatureCodeGeometry:
			d0.Geometry, err = readGeometryFeature(frdr)
		case FeatureCodeSecureMsg:
			d0.SecureMsg, err = readSecureMsgFeature(frdr)
		case FeatureCodeEnterprise:
			d0.Enterprise, err = readEnterpriseFeature(frdr)
		case FeatureCodeOpalV1:
			d0.OpalV1, err = readOpalV1Feature(frdr)
		case FeatureCodeSingleUser:
			d0.SingleUser, err = readSingleUserFeature(frdr)
		case FeatureCodeDataStore:
			d0.DataStore, err = readDataStoreFeature(frdr)
		case FeatureCodeOpalV2:
			d0.OpalV2, err = readOpalV2Feature(frdr)
		case FeatureCodeOpalite:
			d0.Opalite, err = readOpaliteFeature(frdr)
		case FeatureCodePyriteV1:
			d0.PyriteV1, err = readPyriteV1Feature(frdr)
		case FeatureCodePyriteV2:
			d0.PyriteV2, err = readPyriteV2Feature(frdr)
		case FeatureCodeRubyV1:
			d0.RubyV1, err = readRubyV1Feature(frdr)
		case FeatureCodeLockingLBA:
			d0.LockingLBA, err = readLockingLBAFeature(frdr)
		case FeatureCodeBlockSID:
			d0.BlockSID, err = readBlockSIDFeature(frdr)
		case FeatureCodeNamespaceLocking:
			d0.NamespaceLocking, err = readNamespaceLockingFeature(frdr)
		case FeatureCodeDataRemoval:
			d0.DataRemoval, err = readDataRemovalFeature(frdr)
		case FeatureCodeNamespaceGeometry:
			d0.NamespaceGeometry, err = readNamespaceGeometryFeature(frdr)
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
