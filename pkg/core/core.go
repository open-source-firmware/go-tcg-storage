// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Architecture Core Specification TCG Specification Version 2.01

package core

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/matfax/go-tcg-storage/pkg/drive"
)

// The following needs to be reworked and/or moved. Not sure where to yet.

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

// Request an (extended) ComID.
func GetComID(d drive.DriveIntf) (ComID, error) {
	var comID [512]byte
	comIDs := comID[:]
	if err := d.IFRecv(drive.SecurityProtocolTCGTPer, 0, &comIDs); err != nil {
		return ComIDInvalid, err
	}

	c := binary.BigEndian.Uint16(comID[0:2])
	ce := binary.BigEndian.Uint16(comID[2:4])

	return ComID(uint32(c) + uint32(ce)<<16), nil
}

func HandleComIDRequest(d drive.DriveIntf, comID ComID, req ComIDRequest) ([]byte, error) {
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
func IsComIDValid(d drive.DriveIntf, comID ComID) (bool, error) {
	res, err := HandleComIDRequest(d, comID, ComIDRequestVerifyComIDValid)
	if err != nil {
		return false, err
	}
	state := binary.BigEndian.Uint32(res[0:4])
	return state == 2 || state == 3, nil
}

// Reset the state of the synchronous protocol stack.
func StackReset(d drive.DriveIntf, comID ComID) error {
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

// FindComID checks data of Level0Discovery for the particular SSC and reads the standard ComID
// of requests a ComID if no standard is set.
func FindComID(d drive.DriveIntf, d0 *Level0Discovery) (ComID, ProtocolLevel, error) {
	proto := ProtocolLevelUnknown
	comID := ComIDInvalid
	if d0.OpalV2 != nil {
		comID = ComID(d0.OpalV2.BaseComID)
		proto = ProtocolLevelCore
	} else if d0.PyriteV1 != nil {
		comID = ComID(d0.PyriteV1.BaseComID)
		proto = ProtocolLevelCore
	} else if d0.PyriteV2 != nil {
		comID = ComID(d0.PyriteV2.BaseComID)
		proto = ProtocolLevelCore
	} else if d0.Enterprise != nil {
		comID = ComID(d0.Enterprise.BaseComID)
		proto = ProtocolLevelEnterprise
	} else if d0.RubyV1 != nil {
		comID = ComID(d0.RubyV1.BaseComID)
		proto = ProtocolLevelCore
	}

	autoComID, err := GetComID(d)
	if err == nil && autoComID > 0 {
		comID = autoComID
	}

	return comID, proto, nil
}
