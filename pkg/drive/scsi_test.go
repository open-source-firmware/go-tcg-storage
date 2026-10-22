// Copyright (c) 2026 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drive

import (
	"bytes"
	"errors"
	"testing"
)

const checkCondition = 0x02 // SCSI status

// vpd encodes a VPD page with the given contents.
func vpd(page uint8, contents ...byte) []byte {
	l := len(contents)
	return append([]byte{0x00, page, byte(l >> 8), byte(l)}, contents...)
}

func cat(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// Identification descriptors: a NAA designator of the logical unit without a
// protocol identifier, and of the target port with the SAS one
var (
	luDesignator   = []byte{0x01, 0x03, 0x00, 0x08, 0x50, 0x00, 0xc5, 0x00, 0x12, 0x34, 0x56, 0x78}
	portDesignator = []byte{0x61, 0x93, 0x00, 0x08, 0x50, 0x00, 0xc5, 0x00, 0x12, 0x34, 0x56, 0x79}
)

// fakeSCSITransport answers INQUIRY commands from canned responses, keyed by
// VPD page, or -1 for the standard INQUIRY data, and records all CDBs.
type fakeSCSITransport struct {
	inquiry map[int][]byte
	result  SCSIResult // returned for all other commands
	cdbs    [][]byte
}

func (f *fakeSCSITransport) SendCDB(cdb []byte, dir SCSIDirection, data []byte) (SCSIResult, error) {
	f.cdbs = append(f.cdbs, bytes.Clone(cdb))
	if cdb[0] != scsiOpInquiry {
		return f.result, nil
	}
	key := -1
	if cdb[1]&0x01 != 0 {
		key = int(cdb[2])
	}
	resp, ok := f.inquiry[key]
	if !ok {
		// CHECK CONDITION, ILLEGAL REQUEST
		return SCSIResult{Status: checkCondition, Sense: []byte{0x70, 0x00, 0x05}}, nil
	}
	copy(data, resp)
	return SCSIResult{Status: scsiStatusGood}, nil
}

func (f *fakeSCSITransport) Close() error {
	return nil
}

// A SATA drive, behind the SCSI/ATA Translation layer of Linux libata.
func TestSCSIDriveIdentifySATA(t *testing.T) {
	// Linux libata reports the full 20 byte ATA serial number, including
	// padding, with a page length of 20.
	serial := []byte("S3Z1NB0K123456X     ")
	f := &fakeSCSITransport{inquiry: map[int][]byte{
		// Standard INQUIRY data, as emulated by Linux libata
		-1: cat([]byte{0x00, 0x00, 0x05, 0x02, 0x1f, 0x00, 0x00, 0x00},
			[]byte("ATA     "), []byte("Samsung SSD 860 "), []byte("1B6Q")),
		scsiVPDSupportedPages:       vpd(scsiVPDSupportedPages, 0x00, 0x80, 0x83),
		scsiVPDUnitSerialNumber:     vpd(scsiVPDUnitSerialNumber, serial...),
		scsiVPDDeviceIdentification: vpd(scsiVPDDeviceIdentification, luDesignator...),
	}}
	d := NewSCSIDrive(f)

	id, err := d.Identify()
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	want := Identity{Protocol: "SATA", Model: "Samsung SSD 860", Firmware: "1B6Q", SerialNumber: "S3Z1NB0K123456X"}
	if *id != want {
		t.Errorf("Identify() = %+v, want %+v", *id, want)
	}

	sn, err := d.SerialNumber()
	if err != nil {
		t.Fatalf("SerialNumber() error = %v", err)
	}
	if !bytes.Equal(sn, serial) {
		t.Errorf("SerialNumber() = %q, want %q", sn, serial)
	}
}

// A SAS drive, reporting its transport protocol in the device identification
// VPD page.
func TestSCSIDriveIdentifySAS(t *testing.T) {
	f := &fakeSCSITransport{inquiry: map[int][]byte{
		-1: cat([]byte{0x00, 0x00, 0x06, 0x02, 0x1f, 0x00, 0x00, 0x00},
			[]byte("SEAGATE "), []byte("ST10000NM0608   "), []byte("E001")),
		scsiVPDSupportedPages:       vpd(scsiVPDSupportedPages, 0x00, 0x83),
		scsiVPDDeviceIdentification: vpd(scsiVPDDeviceIdentification, cat(luDesignator, portDesignator)...),
	}}

	id, err := NewSCSIDrive(f).Identify()
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	want := Identity{Protocol: "SAS", Model: "SEAGATE ST10000NM0608", Firmware: "E001"}
	if *id != want {
		t.Errorf("Identify() = %+v, want %+v", *id, want)
	}
}

// Any SCSI drive: SATA drives behind a SCSI/ATA Translation layer get the same
// SECURITY PROTOCOL IN/OUT commands as SAS drives.
func TestSCSIDriveSecurityProtocol(t *testing.T) {
	f := &fakeSCSITransport{}
	d := NewSCSIDrive(f)

	data := make([]byte, 1024)
	if err := d.IFRecv(SecurityProtocolTCGManagement, 0x0007, &data); err != nil {
		t.Fatalf("IFRecv() error = %v", err)
	}
	if err := d.IFSend(SecurityProtocolTCGTPer, 0x1234, data[:512]); err != nil {
		t.Fatalf("IFSend() error = %v", err)
	}
	// INC_512 set, transfer length in 512 byte units
	want := [][]byte{
		{0xa2, 0x01, 0x00, 0x07, 0x80, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00},
		{0xb5, 0x02, 0x12, 0x34, 0x80, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
	}
	for i := range want {
		if !bytes.Equal(f.cdbs[i], want[i]) {
			t.Errorf("CDB %d = % x, want % x", i, f.cdbs[i], want[i])
		}
	}
}

func TestSCSIResultError(t *testing.T) {
	for _, tc := range []struct {
		name         string
		res          SCSIResult
		ok           bool
		notSupported bool
	}{
		{"good", SCSIResult{Status: scsiStatusGood}, true, false},
		{"fixed illegal request", SCSIResult{Status: checkCondition, Sense: []byte{0x70, 0x00, 0x05, 0, 0, 0, 0, 0x0a}}, false, true},
		{"fixed with valid bit", SCSIResult{Status: checkCondition, Sense: []byte{0xf0, 0x00, 0x05}}, false, true},
		{"descriptor illegal request", SCSIResult{Status: checkCondition, Sense: []byte{0x72, 0x05}}, false, true},
		{"not ready", SCSIResult{Status: checkCondition, Sense: []byte{0x70, 0x00, 0x02}}, false, false},
		{"unknown sense format", SCSIResult{Status: checkCondition, Sense: []byte{0x00, 0x00, 0x05}}, false, false},
		{"short sense", SCSIResult{Status: checkCondition, Sense: []byte{0x70, 0x00}}, false, false},
		{"busy", SCSIResult{Status: 0x08}, false, false},
	} {
		err := scsiResultError(tc.res)
		if (err == nil) != tc.ok || errors.Is(err, ErrNotSupported) != tc.notSupported {
			t.Errorf("%s: scsiResultError() = %v", tc.name, err)
		}
	}
}

func TestSCSIVPDPage(t *testing.T) {
	serial := []byte("S3Z1NB0K123456X     ")
	p, err := scsiVPDPage(cat(vpd(scsiVPDUnitSerialNumber, serial...), make([]byte, 64)), scsiVPDUnitSerialNumber)
	if err != nil {
		t.Fatalf("scsiVPDPage() error = %v", err)
	}
	if !bytes.Equal(p, serial) {
		t.Errorf("scsiVPDPage() = %q, want %q", p, serial)
	}

	for _, tc := range []struct {
		name string
		b    []byte
	}{
		{"short", []byte{0x00, 0x80, 0x00}},
		{"wrong page", vpd(scsiVPDDeviceIdentification, 'x')},
		{"empty", vpd(scsiVPDUnitSerialNumber)},
		{"length beyond data", []byte{0x00, 0x80, 0x00, 0x20, 'x', 'y'}},
	} {
		if _, err := scsiVPDPage(tc.b, scsiVPDUnitSerialNumber); !errors.Is(err, ErrUnexpectedResponse) {
			t.Errorf("%s: scsiVPDPage() error = %v, want %v", tc.name, err, ErrUnexpectedResponse)
		}
	}
}

func TestSCSIDeviceIdentificationProtocol(t *testing.T) {
	// Valid protocol identifier, but associated with the logical unit
	luProto := []byte{0x61, 0x83, 0x00, 0x08, 0x50, 0x00, 0xc5, 0x00, 0x12, 0x34, 0x56, 0x7a}

	for _, tc := range []struct {
		name string
		did  []byte
		want scsiProtocol
	}{
		{"SAS port", cat(luDesignator, portDesignator), 6},
		{"no protocol", luDesignator, scsiProtocolUnknown},
		{"logical unit association", luProto, scsiProtocolUnknown},
		// A descriptor claiming more data than present must stop the walk
		{"truncated descriptor", cat(luDesignator, portDesignator[:6]), scsiProtocolUnknown},
		// Trailing bytes too short for a descriptor header are ignored
		{"trailing bytes", cat(portDesignator, []byte{0x00, 0x00}), 6},
	} {
		if got := scsiDeviceIdentificationProtocol(tc.did); got != tc.want {
			t.Errorf("%s: scsiDeviceIdentificationProtocol() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
