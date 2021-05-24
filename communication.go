// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core packetization for communication

package tcgstorage

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bluecmd/go-tcg-storage/drive"
)

var (
	ErrTooLargeComPacket = errors.New("packet assembly constructed a too large ComPacket")
	ErrTooLargePacket    = errors.New("packet assembly constructed a too large Packet")
)

// NOTE: This is almost io.ReadWriter, but not quite - I couldn't figure out
// a good interface use that wouldn't result in a lot of extra copying.
type CommunicationIntf interface {
	Send(proto drive.SecurityProtocol, ses *Session, data []byte) error
	Receive(proto drive.SecurityProtocol, ses *Session) ([]byte, error)
}

type plainCom struct {
	d  DriveIntf
	hp HostProperties
	tp TPerProperties
}

type comPacketHeader struct {
	_               uint32
	ComID           uint16
	ComIDExt        uint16
	OutstandingData uint32
	MinTransfer     uint32
	Length          uint32
}
type packetHeader struct {
	TSN             uint32
	HSN             uint32
	SeqNumber       uint32
	_               uint16
	AckType         uint16
	Acknowledgement uint32
	Length          uint32
}
type subPacketHeader struct {
	_      [6]byte
	Kind   uint16
	Length uint32
}

// Low-level communication used to send/receive packets to a TPer or SP.
//
// Implements Subpacket-Packet-ComPacket packet format.
func NewPlainCommunication(d DriveIntf, hp HostProperties, tp TPerProperties) *plainCom {
	return &plainCom{d, hp, tp}
}

func (c *plainCom) Send(proto drive.SecurityProtocol, ses *Session, data []byte) error {
	// TODO: Packetize
	// From "3.3.10.3 Synchronous Communications Restrictions"
	// > Methods SHALL NOT span ComPackets. In the case where an incomplete method is
	// > submitted, if the TPer is able to identify the associated session, then that session SHALL
	// Maybe add a "fragment" flag to reject too large Sends when synchronous?
	// TODO: Implement fragmentation

	subpkt := bytes.Buffer{}
	spkthdr := subPacketHeader{
		Kind:   0, // Data
		Length: uint32(len(data)),
	}
	if err := binary.Write(&subpkt, binary.BigEndian, &spkthdr); err != nil {
		return err
	}
	subpkt.Write(data)
	pad := 4 - (len(data) % 4)
	subpkt.Write(make([]byte, pad))

	pkt := bytes.Buffer{}
	if uint(pkt.Len()) > c.tp.MaxPacketSize {
		return ErrTooLargePacket
	}
	pkthdr := packetHeader{
		TSN:       uint32(ses.TSN),
		HSN:       uint32(ses.HSN),
		SeqNumber: uint32(ses.SeqLastXmit + 1),
		AckType:   0, /* TODO */
		Length:    uint32(subpkt.Len()),
	}
	if err := binary.Write(&pkt, binary.BigEndian, &pkthdr); err != nil {
		return err
	}
	pkt.Write(subpkt.Bytes())

	compkt := bytes.Buffer{}
	compkthdr := comPacketHeader{
		ComID:           uint16(ses.ComID & 0xffff),
		ComIDExt:        uint16((ses.ComID & 0xffff0000) >> 16),
		OutstandingData: 0, /* Reseved */
		MinTransfer:     0, /* Reserved */
		Length:          uint32(pkt.Len()),
	}
	if err := binary.Write(&compkt, binary.BigEndian, &compkthdr); err != nil {
		return err
	}
	compkt.Write(pkt.Bytes())
	if uint(compkt.Len()) > c.tp.MaxComPacketSize {
		return ErrTooLargeComPacket
	}
	fmt.Printf("com.Send:\n%s\n", hex.Dump(compkt.Bytes()))
	ses.SeqLastXmit += 1
	// Extend buffer to be aligned to 512 byte pages which some drives like
	compkt.Write(make([]byte, 512-(compkt.Len()%512)))
	return c.d.IFSend(proto, uint16(ses.ComID), compkt.Bytes())
}

func (c *plainCom) Receive(proto drive.SecurityProtocol, ses *Session) ([]byte, error) {
	// TODO: Unpacketize
	buf := make([]byte, c.hp.MaxComPacketSize)
	err := c.d.IFRecv(proto, uint16(ses.ComID), &buf)
	if err != nil {
		return nil, err
	}
	fmt.Printf("com.Receive:\n%s\n", hex.Dump(buf))
	return nil, fmt.Errorf("not implemented")
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
