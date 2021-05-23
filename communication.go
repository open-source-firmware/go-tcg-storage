// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core packetization for communication

package tcgstorage

import (
	"github.com/bluecmd/go-tcg-storage/drive"
)

// NOTE: This is almost io.ReadWriter, but not quite - I couldn't figure out
// a good interface use that wouldn't result in a lot of extra copying.
type CommunicationIntf interface {
	Send(proto drive.SecurityProtocol, ses *Session, data []byte) error
	Receive(proto drive.SecurityProtocol, ses *Session) ([]byte, error)
}

type plainCom struct {
	d DriveIntf
}

// Low-level communication used to send/receive packets to a TPer or SP.
//
// Implements Subpacket-Packet-ComPacket packet format.
func NewPlainCommunication(d DriveIntf) *plainCom {
	return &plainCom{d}
}

func (s *plainCom) Send(proto drive.SecurityProtocol, ses *Session, data []byte) error {
	// TODO: Packetize
	return s.d.IFSend(proto, uint16(ses.ComID), data)
}

func (s *plainCom) Receive(proto drive.SecurityProtocol, ses *Session) ([]byte, error) {
	// TODO: Unpacketize
	buf := make([]byte, 1024)
	err := s.d.IFRecv(proto, uint16(ses.ComID), &buf)
	return buf, err
}

//  Create header:
// typedef struct _OPALComPacket {
//     uint32_t reserved0;
//     uint8_t extendedComID[4];
/*
Extended ComID for static ComIDs:
hdr->cp.extendedComID[0] = ((comID & 0xff00) >> 8);
hdr->cp.extendedComID[1] = (comID & 0x00ff);
hdr->cp.extendedComID[2] = 0x00;
hdr->cp.extendedComID[3] = 0x00;
*/
//     uint32_t outstandingData;
//     uint32_t minTransfer;
//     uint32_t length;
// } OPALComPacket;
//
// /** Packet structure. */
// typedef struct _OPALPacket {
//     uint32_t TSN;
//     uint32_t HSN;
//     uint32_t seqNumber;
//     uint16_t reserved0;
//     uint16_t ackType;
//     uint32_t acknowledgement;
//     uint32_t length;
// } OPALPacket;
// 
// /** Data sub packet header */
// typedef struct _OPALDataSubPacket {
//     uint8_t reserved0[6];
//     uint16_t kind;
//     uint32_t length;
// } OPALDataSubPacket;

//  typedef struct _OPALHeader {
//    OPALComPacket cp;
//    OPALPacket pkt;
//    OPALDataSubPacket subpkt;
//  } OPALHeader;
//  hdr->subpkt.length = SWAP32(bufferpos - (sizeof (OPALHeader)));
//  Pad for 4 byte alignment:
//  while (bufferpos % 4 != 0) {
//    cmdbuf[bufferpos++] = 0x00;
//  }
// hdr->pkt.length = SWAP32((bufferpos - sizeof (OPALComPacket))
//                       - sizeof (OPALPacket));
// hdr->cp.length = SWAP32(bufferpos - sizeof (OPALComPacket));

// Length
// The field identifies the number of bytes in the Data portion of the subpacket Payload. This value does
// not include the length of the Pad portion of the Payload.
// The pad field ensures that the boundaries between subpackets (and therefore packets) are aligned to
// 4-byte boundaries. The number of pad bytes SHALL be (-Subpacket.Length modulo 4). This field
// SHALL be zeroes
