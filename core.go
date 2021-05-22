// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01
// for getting access to OPAL 2.0 functions.

package opal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bluecmd/go-opal/drive"
)

type DriveIntf interface {
	IFRecv(proto drive.SecurityProtocol, comID drive.ComID, data *[]byte) error
	IFSend(proto drive.SecurityProtocol, comID drive.ComID, data *[]byte) error
}

type FeatureCode uint16

const (
	FeatureTPer    FeatureCode = 0x0001
	FeatureLocking FeatureCode = 0x0002
	FeatureOPAL20  FeatureCode = 0x0203
)

type TPerFeature struct {
	// TODO
}

type LockingFeature struct {
	// TODO
}

type OPAL20Feature struct {
	// TODO
}

type Level0Discovery struct {
	MajorVersion int
	MinorVersion int
	Vendor [32]byte
	Locking *LockingFeature
	TPer    *TPerFeature
	OPAL20  *OPAL20Feature
}

func Discovery0(d DriveIntf) (*Level0Discovery, error) {
	d0raw := make([]byte, 2048)
	if err := d.IFRecv(drive.SecurityProtocolManagement, drive.ComIDDiscoveryL0, &d0raw); err != nil {
		return nil, err
	}
	d0 := &Level0Discovery{}
	d0buf := bytes.NewBuffer(d0raw)
	d0hdr := struct {
		Size uint32
		Major uint16
		Minor uint16
		_ [8]byte
		Vendor [32]byte
	}{}
	if err := binary.Read(d0buf, binary.BigEndian, &d0hdr); err != nil {
		return nil, fmt.Errorf("Failed to parse Level 0 discovery: %v", err)
	}
	d0.MajorVersion = int(d0hdr.Major)
	d0.MinorVersion = int(d0hdr.Minor)
	copy(d0.Vendor[:], d0hdr.Vendor[:])

	fsize := int(d0hdr.Size) - binary.Size(d0hdr) + 4
	for fsize > 0 {
		fhdr := struct {
			Code FeatureCode
			Version uint8
			Size uint8
		}{}
		if err := binary.Read(d0buf, binary.BigEndian, &fhdr); err != nil {
			return nil, fmt.Errorf("Failed to parse feature header: %v", err)
		}
		frdr := io.LimitReader(d0buf, int64(fhdr.Size))
		var err error
		switch (fhdr.Code) {
		case FeatureTPer:
			d0.TPer, err = readTPerFeature(frdr)
		case FeatureLocking:
			d0.Locking, err = readLockingFeature(frdr)
		case FeatureOPAL20:
			d0.OPAL20, err = readOPAL20Feature(frdr)
		default:
			// Unsupported feature
		}
		if err != nil {
			return nil, err
		}
		io.CopyN(ioutil.Discard, frdr, int64(fhdr.Size))
		fsize -= binary.Size(fhdr) + int(fhdr.Size)
	}
	return d0, nil
}

func readTPerFeature(rdr io.Reader) (*TPerFeature, error) {
	f := &TPerFeature{}
	return f, nil
}

func readLockingFeature(rdr io.Reader) (*LockingFeature, error) {
	f := &LockingFeature{}
	return f, nil
}

func readOPAL20Feature(rdr io.Reader) (*OPAL20Feature, error) {
	f := &OPAL20Feature{}
	return f, nil
}
