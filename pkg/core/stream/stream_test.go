// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests implementation of TCG Storage Core Data Stream

package stream

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestBytes(t *testing.T) {
	testCases := []struct {
		name string
		data string
		want string
	}{
		{"Null", "", "A0"},
		{"Tiny byte", "2F", "A1 2F"}, // 3.2.2.3.1 Simple Tokens – Atoms Overview ("Tiny atoms only represent integers")
		{"Short byte", "8F", "A1 8F"},
		{"8 bytes", "01 02 03 04 05 06 07 08", "A8 01 02 03 04 05 06 07 08"},
		{"60 bytes",
			"464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152",
			"d03c464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152464f4f424152",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, _ := hex.DecodeString(strings.ReplaceAll(tc.data, " ", ""))
			want, _ := hex.DecodeString(strings.ReplaceAll(tc.want, " ", ""))
			if got := Bytes(in); !bytes.Equal(got, want) {
				t.Errorf("In(%+v) = %+v; want %+v", in, got, want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		name string
		data string
		want List
		err  error
	}{
		{"Null", "A0", List{[]byte{}}, nil},
		{"Call", "F8", List{Call}, nil},
		{"Tiny byte", "A1 2F", List{[]byte{0x2f}}, nil},
		{"Tiny uint", "2F", List{uint(0x2f)}, nil},
		{"Short byte", "A1 8F", List{[]byte{0x8f}}, nil},
		{"Short uint", "81 8F", List{uint(0x8f)}, nil},
		{"8 bytes", "A8 01 02 03 04 05 06 07 08", List{[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, nil},
		{"16 bytes", "D0 10 01 02 03 04 05 06 07 08 01 02 03 04 05 06 07 08",
			List{[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, nil},
		{"Long byte", "E2 00 00 04 01 02 03 04", List{[]byte{0x01, 0x02, 0x03, 0x04}}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, _ := hex.DecodeString(strings.ReplaceAll(tc.data, " ", ""))
			if got, err := Decode(in); !reflect.DeepEqual(got, tc.want) || err != tc.err {
				t.Errorf("In(%+v) = %+v, %+v; want %+v, %+v", in, got, err, tc.want, tc.err)
			}
		})
	}
}

func TestDecodeLists(t *testing.T) {
	testCases := []struct {
		name string
		data string
		want List
		err  error
	}{
		{"Bad list", "F1", nil, ErrUnbalancedList},
		{"Empty list", "F0 F1", List{List{}}, nil},
		{"One element", "F0 F8 F1", List{List{Call}}, nil},
		{"Two nested element", "F0 F0 F8 F8 F1 F1", List{List{List{Call, Call}}}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, _ := hex.DecodeString(strings.ReplaceAll(tc.data, " ", ""))
			if got, err := Decode(in); !reflect.DeepEqual(got, tc.want) || err != tc.err {
				t.Errorf("In(%+v) = %+v, %+v; want %+v, %+v", in, got, err, tc.want, tc.err)
			}
		})
	}

}
