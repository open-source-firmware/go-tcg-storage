// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

// fakeDrive implements drive.DriveIntf with a canned IFRecv response or
// error. Only IFRecv carries behaviour; the remaining methods exist purely
// to satisfy the interface.
type fakeDrive struct {
	recv []byte
	err  error
}

func (f *fakeDrive) IFRecv(_ drive.SecurityProtocol, _ uint16, data *[]byte) error {
	if f.err != nil {
		return f.err
	}
	// Mirror nvme_nix.go / scsi_nix.go: write into the caller-provided
	// backing slice without resizing it.
	copy(*data, f.recv)
	return nil
}

func (f *fakeDrive) IFSend(drive.SecurityProtocol, uint16, []byte) error { return nil }
func (f *fakeDrive) Identify() (*drive.Identity, error)                  { return nil, nil }
func (f *fakeDrive) SerialNumber() ([]byte, error)                       { return nil, nil }
func (f *fakeDrive) Close() error                                        { return nil }

// buildWire encodes a full ComPacket/Packet/SubPacket header stack followed
// by an optional payload. Length fields are written verbatim so the test
// code can construct arbitrary (including malformed) responses.
func buildWire(t *testing.T, comLen, pktLen uint32, subKind uint16, subLen uint32, payload []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, comPacketHeader{Length: comLen}); err != nil {
		t.Fatalf("encode comPacketHeader: %v", err)
	}
	if err := binary.Write(&b, binary.BigEndian, packetHeader{Length: pktLen}); err != nil {
		t.Fatalf("encode packetHeader: %v", err)
	}
	if err := binary.Write(&b, binary.BigEndian, subPacketHeader{Kind: subKind, Length: subLen}); err != nil {
		t.Fatalf("encode subPacketHeader: %v", err)
	}
	b.Write(payload)
	return b.Bytes()
}

func newTestCom(d drive.DriveIntf, maxCom, maxPkt uint) *plainCom {
	return &plainCom{
		d: d,
		hp: HostProperties{
			MaxComPacketSize: maxCom,
			MaxPacketSize:    maxPkt,
		},
	}
}

// TestReceive_IFRecvError verifies that errors from the underlying driver
// are propagated untouched.
func TestReceive_IFRecvError(t *testing.T) {
	sentinel := errors.New("driver boom")
	c := newTestCom(&fakeDrive{err: sentinel}, 2048, 1024)
	_, err := c.Receive(&Session{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

// TestReceive_HappyPath exercises the standard data-subpacket flow.
func TestReceive_HappyPath(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	wire := buildWire(t, 40, 16, 0, uint32(len(payload)), payload)
	c := newTestCom(&fakeDrive{recv: wire}, 2048, 1024)
	resp, err := c.Receive(&Session{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bytes.Equal(resp, payload) {
		t.Fatalf("resp = %x, want %x", resp, payload)
	}
}

// TestReceive_TooLargeComPacket triggers the ErrTooLargeComPacket guard.
func TestReceive_TooLargeComPacket(t *testing.T) {
	wire := buildWire(t, 999_999, 0, 0, 0, nil)
	c := newTestCom(&fakeDrive{recv: wire}, 2048, 1024)
	_, err := c.Receive(&Session{})
	if !errors.Is(err, ErrTooLargeComPacket) {
		t.Fatalf("err = %v, want %v", err, ErrTooLargeComPacket)
	}
}

// TestReceive_TooLargePacket triggers the ErrTooLargePacket guard.
func TestReceive_TooLargePacket(t *testing.T) {
	wire := buildWire(t, 40, 999_999, 0, 0, nil)
	c := newTestCom(&fakeDrive{recv: wire}, 2048, 1024)
	_, err := c.Receive(&Session{})
	if !errors.Is(err, ErrTooLargePacket) {
		t.Fatalf("err = %v, want %v", err, ErrTooLargePacket)
	}
}

// TestReceive_NonDataSubpacketKind rejects non-data subpacket kinds with a
// descriptive error. Buffer-management subpackets are not implemented.
func TestReceive_NonDataSubpacketKind(t *testing.T) {
	wire := buildWire(t, 40, 16, 1, 4, []byte{1, 2, 3, 4})
	c := newTestCom(&fakeDrive{recv: wire}, 2048, 1024)
	_, err := c.Receive(&Session{})
	if err == nil || !strings.Contains(err.Error(), "only data subpackets") {
		t.Fatalf("err = %v, want substring %q", err, "only data subpackets")
	}
}

// TestReceive_TruncatedHeaders exercises the three binary.Read error paths
// by sizing MaxComPacketSize below the offset of each successive header.
// These HostProperties values are not realistic in production but the
// parser must still propagate the read error rather than panic.
//
// Sizes: comPacketHeader=20, packetHeader=24, subPacketHeader=12.
func TestReceive_TruncatedHeaders(t *testing.T) {
	cases := []struct {
		name string
		// wire shapes the prefix bytes that survive copy() into the
		// undersized buffer. Only the fields up to the truncation point
		// are observed by the parser.
		wire []byte
		// maxCom controls both buf length and the MaxComPacketSize guard.
		maxCom uint
	}{
		{
			name:   "truncated within comPacketHeader",
			wire:   nil, // first 10 bytes of buf are zero; parser fails before reading Length
			maxCom: 10,
		},
		{
			name:   "truncated within packetHeader",
			wire:   buildWire(t, 10, 0, 0, 0, nil), // compkthdr.Length=10 ≤ maxCom, non-zero
			maxCom: 25,
		},
		{
			name:   "truncated within subPacketHeader",
			wire:   buildWire(t, 30, 12, 0, 0, nil), // both upper Lengths non-zero, ≤ limits
			maxCom: 50,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestCom(&fakeDrive{recv: tc.wire}, tc.maxCom, 1024)
			_, err := c.Receive(&Session{})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			// binary.Read may return either io.EOF (clean) or
			// io.ErrUnexpectedEOF (mid-struct), depending on alignment.
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("err = %v, want io.EOF or io.ErrUnexpectedEOF", err)
			}
		})
	}
}
