// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core packetization for communication

package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/matfax/go-tcg-storage/pkg/drive"
)

var (
	ErrTooLargeComPacket = errors.New("encountered a too large ComPacket")
	ErrTooLargePacket    = errors.New("encountered a too large Packet")
)

// NOTE: This is almost io.ReadWriter, but not quite - I couldn't figure out
// a good interface use that wouldn't result in a lot of extra copying.
type CommunicationIntf interface {
	Send(ses *Session, data []byte) error
	Receive(ses *Session) ([]byte, error)
}

type plainCom struct {
	d  drive.DriveIntf
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
func NewPlainCommunication(d drive.DriveIntf, hp HostProperties, tp TPerProperties) *plainCom {
	return &plainCom{d, hp, tp}
}

func (c *plainCom) Send(ses *Session, data []byte) error {
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
	if (len(data) % 4) > 0 {
		pad := 4 - (len(data) % 4)
		subpkt.Write(make([]byte, pad))
	}

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
	if !c.tp.SequenceNumbers || !c.hp.SequenceNumbers {
		pkthdr.SeqNumber = 0
	}
	if err := binary.Write(&pkt, binary.BigEndian, &pkthdr); err != nil {
		return err
	}
	pkt.Write(subpkt.Bytes())

	compkt := bytes.Buffer{}
	compkthdr := comPacketHeader{
		ComID:           uint16(ses.ComID & 0xffff),
		ComIDExt:        uint16((ses.ComID & 0xffff0000) >> 16),
		OutstandingData: 0, /* Reserved */
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
	if c.tp.SequenceNumbers && c.hp.SequenceNumbers {
		ses.SeqLastXmit += 1
	}
	// Extend buffer to be aligned to 512 byte pages which some drives like
	compkt.Write(make([]byte, 512-(compkt.Len()%512)))
	return c.d.IFSend(drive.SecurityProtocolTCGManagement, uint16(ses.ComID), compkt.Bytes())
}

func (c *plainCom) Receive(ses *Session) ([]byte, error) {
	buf := make([]byte, c.hp.MaxComPacketSize)
	if err := c.d.IFRecv(drive.SecurityProtocolTCGManagement, uint16(ses.ComID), &buf); err != nil {
		return nil, err
	}
	rdr := bytes.NewBuffer(buf)
	compkthdr := comPacketHeader{}
	if err := binary.Read(rdr, binary.BigEndian, &compkthdr); err != nil {
		return nil, err
	}
	if uint(compkthdr.Length) > c.hp.MaxComPacketSize {
		return nil, ErrTooLargeComPacket
	}
	// TODO: Handle OutstandingData and MinTransfer (if needed, haven't checked)
	pkthdr := packetHeader{}
	if err := binary.Read(rdr, binary.BigEndian, &pkthdr); err != nil {
		return nil, err
	}
	if uint(pkthdr.Length) > c.hp.MaxPacketSize {
		return nil, ErrTooLargePacket
	}
	// TODO: Handle SeqNumber
	// TODO: Handle AckType
	subpkthdr := subPacketHeader{}
	if err := binary.Read(rdr, binary.BigEndian, &subpkthdr); err != nil {
		return nil, err
	}
	// TODO: Implement buffer management
	if subpkthdr.Kind != 0 {
		return nil, fmt.Errorf("only data subpackets are implemented")
	}
	data := rdr.Bytes()
	data = data[0:subpkthdr.Length]
	return data, nil
}
