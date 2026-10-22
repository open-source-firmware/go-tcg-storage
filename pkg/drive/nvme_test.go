// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"errors"
	"testing"
)

func nvmeIdentifyData(serial, model, firmware string) []byte {
	b := make([]byte, nvmeIdentifyLength)
	copy(b[4:], serial)
	copy(b[24:], model)
	copy(b[64:], firmware)
	return b
}

// fakeNVMeTransport answers Identify with canned data, returns a fixed status
// for all other commands, and records all commands.
type fakeNVMeTransport struct {
	identify []byte
	status   NVMeStatus
	cmds     []NVMeAdminCommand
}

func (f *fakeNVMeTransport) AdminCommand(cmd NVMeAdminCommand, data []byte) (NVMeStatus, error) {
	f.cmds = append(f.cmds, cmd)
	if cmd.Opcode == nvmeOpIdentify {
		copy(data, f.identify)
		return 0, nil
	}
	return f.status, nil
}

func (f *fakeNVMeTransport) Close() error {
	return nil
}

func TestNVMeDriveIdentify(t *testing.T) {
	// Model and firmware as reported by a Samsung SSD 970 EVO Plus
	f := &fakeNVMeTransport{identify: nvmeIdentifyData("SERIAL0123456789    ", "Samsung SSD 970 EVO Plus 250GB          ", "1B2QEXM7")}
	d := NewNVMeDrive(f)

	id, err := d.Identify()
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	want := Identity{Protocol: "NVMe", Model: "Samsung SSD 970 EVO Plus 250GB", Firmware: "1B2QEXM7", SerialNumber: "SERIAL0123456789"}
	if *id != want {
		t.Errorf("Identify() = %+v, want %+v", *id, want)
	}

	// The full serial number field, including padding
	sn, err := d.SerialNumber()
	if err != nil {
		t.Fatalf("SerialNumber() error = %v", err)
	}
	if !bytes.Equal(sn, []byte("SERIAL0123456789    ")) {
		t.Errorf("SerialNumber() = %q", sn)
	}

	// Identify Controller, not a namespace
	if f.cmds[0] != (NVMeAdminCommand{Opcode: 0x06, CDW10: 0x01}) {
		t.Errorf("sent %+v, want Identify Controller", f.cmds[0])
	}
}

func TestNVMeDriveSecurity(t *testing.T) {
	f := &fakeNVMeTransport{}
	d := NewNVMeDrive(f)

	data := make([]byte, 2048)
	if err := d.IFRecv(SecurityProtocolTCGManagement, 0x07fe, &data); err != nil {
		t.Fatalf("IFRecv() error = %v", err)
	}
	if err := d.IFSend(SecurityProtocolTCGTPer, 0x0001, data[:512]); err != nil {
		t.Fatalf("IFSend() error = %v", err)
	}
	want := []NVMeAdminCommand{
		{Opcode: 0x82, CDW10: 0x0107fe00, CDW11: 2048},
		{Opcode: 0x81, CDW10: 0x02000100, CDW11: 512},
	}
	for i := range want {
		if f.cmds[i] != want[i] {
			t.Errorf("command %d = %+v, want %+v", i, f.cmds[i], want[i])
		}
	}

	for _, tc := range []struct {
		name   string
		status NVMeStatus
		want   error
	}{
		{"Invalid Command Opcode", 0x0001, ErrNotSupported},
		{"Invalid Field in Command with Do Not Retry", 0x4002, ErrNotSupported},
		{"Internal Error", 0x0006, NVMeStatus(0x0006)},
	} {
		f.status = tc.status
		if err := d.IFRecv(SecurityProtocolTCGManagement, 0, &data); !errors.Is(err, tc.want) {
			t.Errorf("%s: IFRecv() error = %v, want %v", tc.name, err, tc.want)
		}
	}
}
