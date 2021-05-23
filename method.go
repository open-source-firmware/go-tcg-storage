// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package tcgstorage

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/drive"
)

type InvokingID [8]byte
type MethodID [8]byte

var (
	InvokeIDSMU InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}
	// 00 00 00 00 00 00 00 00 Used to represent null uid
	// 00 00 00 00 00 00 00 01 Used as the SPUID, the UID that identifies "This SP" – used as the InvokingID for invocation of SP methods
	// 00 00 00 00 00 00 00 FF Used as the SMUID, the UID that identifies "the Session manager" – used as InvokingID for invocation of Session Manager layer methods
	// 00 00 00 00 00 00 FF xx Identifies UIDs assigned to Session Manager layer methods, where xx is the UID assigned to a particular method (see Table 241)
	// 00 00 00 0B 00 00 00 01 Used in the C_PIN table's CharSet column to indicate that the GenKey character set is not restricted (all byte values are legal).

	// Table 241 Session Manager Method UIDs
	// Method UID Method Name
	MethodIDProperties MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x01}

// 00 00 00 00 00 00 FF 01 Properties
// 00 00 00 00 00 00 FF 02 StartSession
// 00 00 00 00 00 00 FF 03 SyncSession
// 00 00 00 00 00 00 FF 04 StartTrustedSession
// 00 00 00 00 00 00 FF 05 SyncTrustedSession
// 00 00 00 00 00 00 FF 06 CloseSession

// Table 242 MethodID UIDs
// UID in MethodID Table Method Name Template
// 00 00 00 06 00 00 00 01 DeleteSP Base
// 00 00 00 06 00 00 00 02 CreateTable Base
// 00 00 00 06 00 00 00 03 Delete Base
// 00 00 00 06 00 00 00 04 CreateRow Base
// 00 00 00 06 00 00 00 05 DeleteRow Base
// 00 00 00 06 00 00 00 06 OBSOLETE *
// 00 00 00 06 00 00 00 07 OBSOLETE *
// 00 00 00 06 00 00 00 08 Next Base
// 00 00 00 06 00 00 00 09 GetFreeSpace Base
// 00 00 00 06 00 00 00 0A GetFreeRows Base
// 00 00 00 06 00 00 00 0B DeleteMethod Base
// 00 00 00 06 00 00 00 0C OBSOLETE *
// 00 00 00 06 00 00 00 0D GetACL Base
// 00 00 00 06 00 00 00 0E AddACE Base
// 00 00 00 06 00 00 00 0F RemoveACE Base
// 00 00 00 06 00 00 00 10 GenKey Base
// 00 00 00 06 00 00 00 11 Reserved for SSC Usage
// 00 00 00 06 00 00 00 12 GetPackage Base
// 00 00 00 06 00 00 00 13 SetPackage Base
// 00 00 00 06 00 00 00 16 Get Base
// 00 00 00 06 00 00 00 17 Set Base
// 00 00 00 06 00 00 00 1C Authenticate Base
// 00 00 00 06 00 00 02 01 IssueSP Admin
// 00 00 00 06 00 00 02 02 Reserved for SSC Usage
// 00 00 00 06 00 00 02 03 Reserved for SSC Usage
// 00 00 00 06 00 00 04 01 GetClock Clock
// 00 00 00 06 00 00 04 02 ResetClock Clock
// 00 00 00 06 00 00 04 03 SetClockHigh Clock
// 00 00 00 06 00 00 04 04 SetLagHigh Clock
// 00 00 00 06 00 00 04 05 SetClockLow Clock
// 00 00 00 06 00 00 04 06 SetLagLow Clock
// 00 00 00 06 00 00 04 07 IncrementCounter Clock
// 00 00 00 06 00 00 06 01 Random Crypto
// 00 00 00 06 00 00 06 02 Salt Crypto
// 00 00 00 06 00 00 06 03 DecryptInit Crypto
// 00 00 00 06 00 00 06 04 Decrypt Crypto
// 00 00 00 06 00 00 06 05 DecryptFinalize Crypto
// 00 00 00 06 00 00 06 06 EncryptInit Crypto
// 00 00 00 06 00 00 06 07 Encrypt Crypto
// 00 00 00 06 00 00 06 08 EncryptFinalize Crypto
// 00 00 00 06 00 00 06 09 HMACInit Crypto
// 00 00 00 06 00 00 06 0A HMAC Crypto
// 00 00 00 06 00 00 06 0B HMACFinalize Crypto
// 00 00 00 06 00 00 06 0C HashInit Crypto
// 00 00 00 06 00 00 06 0D Hash Crypto
// 00 00 00 06 00 00 06 0E HashFinalize Crypto
// 00 00 00 06 00 00 06 0F Sign Crypto
// 00 00 00 06 00 00 06 10 Verify Crypto
// 00 00 00 06 00 00 06 11 XOR Crypto
// 00 00 00 06 00 00 0A 01 AddLog Log
// 00 00 00 06 00 00 0A 02 CreateLog Log
// 00 00 00 06 00 00 0A 03 ClearLog Log
// 00 00 00 06 00 00 0A 04 FlushLog Log
// 00 00 00 06 00 00 08 03 Reserved for SSC Usage

)

type MethodCall struct {
	iuid InvokingID
	muid MethodID
}

func NewMethodCall(iid InvokingID, mid MethodID) *MethodCall {
	return &MethodCall{iid, mid}
}

func (m *MethodCall) MarshalBinary() ([]byte, error) {
	// Session Manager method calls are written as follows, where
	// “SMUID” is 0x00 0x00 0x00 0x00
	// 0x00 0x00 0x00 0xFF: SMUID.MethodName[ <Parameters> ]

	// SP method calls are written as follows, where
	// "ThisSP" is 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x01:
	// ThisSP.MethodName[ <Parameters> ]

	// Table method calls are written as follows, where TableUID is the UID of the table (see 3.2.5.3)
	// upon which the method is being invoked: TableUID.MethodName[ <Parameters> ]
	// d. Object methods are written as follows, where ObjectUID is the UID of the object (see 3.2.5.3)
	// upon which the method is being invoked: ObjectUID.MethodName[ <Parameters> ]

	// The InvokingID (the table or object UID to which the method applies) and MethodName (the method's
	// UID column value as it is defined in an SP's MethodID table) appear first in the signature. The
	// parameters follow, enclosed in "list" delimiters ("[" and "]"). T

	// bufferpos = sizeof(OPALHeader)
	// cmdbuf[bufferpos++] = OPAL_TOKEN::CALL;
	// memcpy(&cmdbuf[bufferpos], &OPALMETHOD[InvokingUID][0], 8);·
	// cmdbuf[bufferpos++] = OPAL_SHORT_ATOM::BYTESTRING8;
	// memcpy(&cmdbuf[bufferpos], &OPALMETHOD[method][0], 8);
	// more data
	// sendCommand(HSN=0,TSN=0, ComID = Opal ComId (or e.g. disk_info.Enterprise_basecomID))
	// HSN = response.getUint32(4) (bigendian)
	// TSN = response.getUint32(5) (bigendian)

	//  cmdbuf[bufferpos++] = OPAL_TOKEN::ENDOFDATA; // End of method call
	//  cmdbuf[bufferpos++] = OPAL_TOKEN::STARTLIST; // Start of Status code list
	//  cmdbuf[bufferpos++] = 0x00; // Expected status code (used for aborts)
	//  cmdbuf[bufferpos++] = 0x00; // Reserved
	//  cmdbuf[bufferpos++] = 0x00; // Reserved
	//  cmdbuf[bufferpos++] = OPAL_TOKEN::ENDLIST;
	return nil, fmt.Errorf("not implemented")
}

func (m *MethodCall) Execute(c CommunicationIntf, proto drive.SecurityProtocol, ses *Session) ([]byte, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if err = c.Send(proto, ses, b); err != nil {
		return nil, err
	}

	resp, err := c.Receive(proto, ses)
	if err != nil {
		return nil, err
	}
	fmt.Printf("method response: %+v\n", resp)
	return resp, nil
}
