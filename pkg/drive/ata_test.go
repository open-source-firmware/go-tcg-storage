// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"errors"
	"testing"
)

// ataIdentifyData returns IDENTIFY DEVICE data with the given strings, stored
// as the device does: with the characters of each pair swapped.
func ataIdentifyData(serial, model, firmware string) []byte {
	b := make([]byte, ataIdentifyLength)
	for _, f := range []struct {
		off int
		s   string
	}{{20, serial}, {46, firmware}, {54, model}} {
		for i := 0; i < len(f.s); i += 2 {
			b[f.off+i] = f.s[i+1]
			b[f.off+i+1] = f.s[i]
		}
	}
	return b
}

// fakeATATransport answers IDENTIFY DEVICE with canned data, returns a fixed
// result for all other commands, and records all commands.
type fakeATATransport struct {
	identify []byte
	result   ATAResult
	cmds     []ATACommand
}

func (f *fakeATATransport) SendCommand(cmd ATACommand, data []byte) (ATAResult, error) {
	f.cmds = append(f.cmds, cmd)
	if cmd.Command == ataCmdIdentifyDevice {
		copy(data, f.identify)
		return ATAResult{}, nil
	}
	return f.result, nil
}

func (f *fakeATATransport) Close() error {
	return nil
}

func TestATADriveIdentify(t *testing.T) {
	// The same Samsung SSD 860 as in TestSCSIDriveIdentifySATA
	serial := "S3Z1NB0K123456X     "
	f := &fakeATATransport{identify: ataIdentifyData(serial, "Samsung SSD 860 EVO 500GB               ", "RVT04B6Q")}
	d := NewATADrive(f)

	id, err := d.Identify()
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	want := Identity{Protocol: "ATA", Model: "Samsung SSD 860 EVO 500GB", Firmware: "RVT04B6Q", SerialNumber: "S3Z1NB0K123456X"}
	if *id != want {
		t.Errorf("Identify() = %+v, want %+v", *id, want)
	}

	// The full serial number field, including padding, identical to what
	// Linux libata reports in the unit serial number VPD page
	sn, err := d.SerialNumber()
	if err != nil {
		t.Fatalf("SerialNumber() error = %v", err)
	}
	if !bytes.Equal(sn, []byte(serial)) {
		t.Errorf("SerialNumber() = %q, want %q", sn, serial)
	}

	// IDENTIFY DEVICE, sent once as the data is cached
	if len(f.cmds) != 1 || f.cmds[0] != (ATACommand{Protocol: ATAPIODataIn, Command: 0xec}) {
		t.Errorf("sent %+v, want a single IDENTIFY DEVICE", f.cmds)
	}
}

func TestATADriveSecurity(t *testing.T) {
	f := &fakeATATransport{}
	d := NewATADrive(f)

	data := make([]byte, 2048)
	if err := d.IFRecv(SecurityProtocolTCGManagement, 0x07fe, &data); err != nil {
		t.Fatalf("IFRecv() error = %v", err)
	}
	// More than 255 blocks, so that the transfer length spills into LBA (7:0)
	big := make([]byte, 0x0102*512)
	if err := d.IFSend(SecurityProtocolTCGTPer, 0x1234, big); err != nil {
		t.Fatalf("IFSend() error = %v", err)
	}
	want := []ATACommand{
		{Protocol: ATAPIODataIn, Command: 0x5c, Features: 0x01, Count: 0x04, LBA: 0x07fe00},
		{Protocol: ATAPIODataOut, Command: 0x5e, Features: 0x02, Count: 0x02, LBA: 0x123401},
	}
	for i := range want {
		if f.cmds[i] != want[i] {
			t.Errorf("command %d = %+v, want %+v", i, f.cmds[i], want[i])
		}
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"unaligned", make([]byte, 100)},
		{"too large", make([]byte, 0x10000*512)},
		{"empty", nil},
	} {
		n := len(f.cmds)
		if err := d.IFSend(SecurityProtocolTCGTPer, 0, tc.data); err == nil {
			t.Errorf("%s: IFSend() succeeded", tc.name)
		}
		if len(f.cmds) != n {
			t.Errorf("%s: sent %+v", tc.name, f.cmds[n:])
		}
	}
}

func TestATAResultError(t *testing.T) {
	for _, tc := range []struct {
		name         string
		res          ATAResult
		ok           bool
		notSupported bool
	}{
		{"good", ATAResult{Status: 0x50}, true, false},
		{"aborted", ATAResult{Status: 0x51, Error: 0x04}, false, true},
		{"uncorrectable", ATAResult{Status: 0x51, Error: 0x40}, false, false},
		{"device fault", ATAResult{Status: 0x70}, false, false},
		// ABRT is only meaningful with ERR set
		{"device fault with stale error", ATAResult{Status: 0x70, Error: 0x04}, false, false},
	} {
		err := ataResultError(tc.res)
		if (err == nil) != tc.ok || errors.Is(err, ErrNotSupported) != tc.notSupported {
			t.Errorf("%s: ataResultError() = %v", tc.name, err)
		}
	}
}
