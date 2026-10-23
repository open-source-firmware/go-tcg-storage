// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"errors"
	"testing"
)

// recordingSCSITransport returns a fixed result, and records all CDBs and
// their data directions.
type recordingSCSITransport struct {
	result SCSIResult
	cdbs   [][]byte
	dirs   []SCSIDirection
}

func (r *recordingSCSITransport) SendCDB(cdb []byte, dir SCSIDirection, data []byte) (SCSIResult, error) {
	r.cdbs = append(r.cdbs, bytes.Clone(cdb))
	r.dirs = append(r.dirs, dir)
	return r.result, nil
}

func (r *recordingSCSITransport) Close() error {
	return nil
}

func TestSATTransportCDB(t *testing.T) {
	r := &recordingSCSITransport{}
	d := NewATADrive(NewSATTransport(r))

	data := make([]byte, 2048)
	if err := d.IFRecv(SecurityProtocolTCGManagement, 0x07fe, &data); err != nil {
		t.Fatalf("IFRecv() error = %v", err)
	}
	big := make([]byte, 0x0102*512)
	if err := d.IFSend(SecurityProtocolTCGTPer, 0x1234, big); err != nil {
		t.Fatalf("IFSend() error = %v", err)
	}
	if _, err := d.Identify(); err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	// A 48-bit command, which needs EXTEND
	ext := ATACommand{Protocol: ATAPIODataIn, Command: 0x2f, Count: 0x0100, LBA: 0x0000_0000_0030}
	if _, err := NewSATTransport(r).SendCommand(ext, data[:512]); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}

	want := []struct {
		cdb []byte
		dir SCSIDirection
	}{
		// PIO Data-in, T_DIR from the device, T_LENGTH in the data buffer
		{[]byte{0x85, 0x08, 0x0b, 0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0xfe, 0x00, 0x07, 0x00, 0x5c, 0x00}, SCSIDataIn},
		{[]byte{0x85, 0x0a, 0x03, 0x00, 0x02, 0x00, 0x02, 0x00, 0x01, 0x00, 0x34, 0x00, 0x12, 0x00, 0x5e, 0x00}, SCSIDataOut},
		{[]byte{0x85, 0x08, 0x0b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xec, 0x00}, SCSIDataIn},
		{[]byte{0x85, 0x09, 0x0b, 0x00, 0x00, 0x01, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2f, 0x00}, SCSIDataIn},
	}
	for i := range want {
		if !bytes.Equal(r.cdbs[i], want[i].cdb) || r.dirs[i] != want[i].dir {
			t.Errorf("CDB %d = % x (%d), want % x (%d)", i, r.cdbs[i], r.dirs[i], want[i].cdb, want[i].dir)
		}
	}
}

func TestSATResult(t *testing.T) {
	// ATA Status Return descriptor of an aborted command, after another
	// descriptor, as Linux libata reports it in descriptor format sense data
	info := []byte{0x00, 0x0a, 0x80, 0x00, 0, 0, 0, 0, 0, 0, 0, 0}
	ataReturn := []byte{0x09, 0x0c, 0x00, 0x04, 0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0x51}
	desc := cat([]byte{0x72, 0x0b, 0x00, 0x00, 0, 0, 0, byte(len(info) + len(ataReturn))}, info, ataReturn)
	aborted := ATAResult{Status: 0x51, Error: 0x04}

	for _, tc := range []struct {
		name string
		res  SCSIResult
		want ATAResult
		ok   bool
	}{
		{"good", SCSIResult{Status: scsiStatusGood}, ATAResult{}, true},
		{"descriptor", SCSIResult{Status: checkCondition, Sense: desc}, aborted, true},
		{"fixed", SCSIResult{Status: checkCondition, Sense: []byte{0xf0, 0x00, 0x0b, 0x04, 0x51, 0x40, 0x00}}, aborted, true},
		// Rejected by the SAT layer, e.g. Linux libata without allow_tpm
		{"fixed without VALID", SCSIResult{Status: checkCondition, Sense: []byte{0x70, 0x00, 0x05, 0x00, 0x00}}, ATAResult{}, false},
		{"descriptor without ATA Status Return", SCSIResult{Status: checkCondition, Sense: cat([]byte{0x72, 0x05, 0x24, 0x00, 0, 0, 0, byte(len(info))}, info)}, ATAResult{}, false},
		// A descriptor claiming more data than present must stop the walk
		{"truncated descriptor", SCSIResult{Status: checkCondition, Sense: desc[:len(desc)-1]}, ATAResult{}, false},
		{"no sense", SCSIResult{Status: checkCondition}, ATAResult{}, false},
	} {
		got, err := satResult(tc.res)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("%s: satResult() = %+v, %v, want %+v", tc.name, got, err, tc.want)
		}
		// Not executing the command is not the device rejecting it
		if errors.Is(err, ErrNotSupported) {
			t.Errorf("%s: satResult() error = %v, must not be %v", tc.name, err, ErrNotSupported)
		}
	}
}
