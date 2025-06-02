// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests implementation of TCG Storage Core Data Stream

package stream

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestTokenType_String(t *testing.T) {
	testCases := []struct {
		name string
		t    TokenType
		want string
	}{
		{"StartList", StartList, "StartList"},
		{"EndList", EndList, "EndList"},
		{"StartName", StartName, "StartName"},
		{"EndName", EndName, "EndName"},
		{"Call", Call, "Call"},
		{"EndOfData", EndOfData, "EndOfData"},
		{"EndOfSession", EndOfSession, "EndOfSession"},
		{"StartTransaction", StartTransaction, "StartTransaction"},
		{"EndTransaction", EndTransaction, "EndTransaction"},
		{"EmptyAtom", EmptyAtom, "EmptyAtom"},
		{"Unknown", 0, "<Unknown>"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.t.String(); got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUInt(t *testing.T) {
	testCases := []struct {
		name string
		data uint
		want []byte
	}{
		{"32", 32, []byte{uint8(32)}},
		{"32768", 32768, []byte{0x82, 0x80, 0x00}},
		{"131072", 131072, []byte{0x84, 0x00, 0x02, 0x00, 0x00}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := UInt(tc.data)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("UInt(%v) = %v; want %v", tc.data, got, tc.want)
			}
		})
	}
}

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
		{"2048 bytes",
			"f85f1dd58ba099d598bd545ad1e467135625749ce37b26faca41a2978c8b2b5774de6574741e5f69f3cc00895ca46cd564ac93bdc278b77fa18cc4c66572f853be5868555421045b32cb1d1e4d8260492ca856c5d1c984b1eae0f8217d4f37a6b2458f77c535d63364deb70a056e888577becefd26fe68301ddb604be6b305e6d6232660562efa93b367292b6304974bf5d4fdfd87a5fccc0e924bc7f82493d23669c8528d2a4a30b3274be34dd0935d4e19da24c92eb74ebcdbac7dfe913f0f3d3bd79d5a3157b2dcd39b273f8e2a0dc730d6d82bf33ec98f0dec97c64440d710f2fd34f685ac7fe3aa4e96f86b997cf0c1884f257c72a98415a606797ab17b24246f122e60ecf438165d63094671558b47d58c2c4229960cbffc41896c72d52345a26324f7f3f80e7eb3217f9ea9352cf480cb00a6ff261e48f6b25c2b8667e8e05b5a565a213a68d2ced44ed0662181f068b58e4e008f1dbffc89dcc4a6bf3c817f35354b0f9668d78c1d176ef657062b76836f90dafbe781e551950f3ecd0606f12ff2e053cdb4708f8c9cf0a9030836b6a20ac31632bae248db19bece98b9a098ba86b2c9efcdb4840efc3b7c515ebf7e5784414bfafbbe0e42c3a0dfeed4ce7d6bf7afa062216d2d856840a0da483f407a0f78934933deac4e685ac2a7edaee227a5134193cebb87906e66183536baec0fb75e30aef292292af1b8062cadc3d10c728129a5602401580d85b31833fba37a5a08d78b4f577c04dd448925b3edda3230a5164d3074d7e8643d2c67463d6a2521283d9c90595bf7877bc603fb5778193df6e2087516c55c7de1835f3018dbdde638004c2410ef16ac9b9a25a68ca79db73cb6674e812f5c0a94dc377a4e77fe0747f92340dfda5bf43d3ff911c902099c5c22ff4eec17f3bdb4710231c3ca0e8df8d71d032ba8b92f60c006c6a17c5a163b2075d7bf73e6b5530f38df26e5782a2ab2e3c10e58fcc9ae5c140051f70e77b7117cf7ec86b4e2abcdabb04300c21105622ea78b9959c6a588973965011d458b74acc81cd86af5c2dbd432f9d61a200fd87c970780bec699aae9f3b2c20be8f3d7f512964308ea4d5645ae21586219d06e39cf169d9e781ddec2fb4559954b7b9639eb70d25d351d9443eb7f0ee54281ef9486bce31920c16344e91848b2bc16766b01ff9e6407c24e28ff5d13dc2cfc6d13c2f3fe3c5e467199d2f52ac7aa8907f4b5a6a2435729f60e7f2754b35a31d209a177c3dce024e3c8b08196ead591d2cbf534fe39017bdd305caa25932ea6acc3acd45020350d3b9e7a3d99ad30576ea17f6a38e7813af4a20eb2dd0892ad57656a91b786b321ad44e78d3ceb36f3e5228fc7817e31005a06f203775b8b06250527adbd37cef65f801891554d1292fcc05c696ac15d9c156383c349e6a500d9f657e77306dcf84bb2f85f1dd58ba099d598bd545ad1e467135625749ce37b26faca41a2978c8b2b5774de6574741e5f69f3cc00895ca46cd564ac93bdc278b77fa18cc4c66572f853be5868555421045b32cb1d1e4d8260492ca856c5d1c984b1eae0f8217d4f37a6b2458f77c535d63364deb70a056e888577becefd26fe68301ddb604be6b305e6d6232660562efa93b367292b6304974bf5d4fdfd87a5fccc0e924bc7f82493d23669c8528d2a4a30b3274be34dd0935d4e19da24c92eb74ebcdbac7dfe913f0f3d3bd79d5a3157b2dcd39b273f8e2a0dc730d6d82bf33ec98f0dec97c64440d710f2fd34f685ac7fe3aa4e96f86b997cf0c1884f257c72a98415a606797ab17b24246f122e60ecf438165d63094671558b47d58c2c4229960cbffc41896c72d52345a26324f7f3f80e7eb3217f9ea9352cf480cb00a6ff261e48f6b25c2b8667e8e05b5a565a213a68d2ced44ed0662181f068b58e4e008f1dbffc89dcc4a6bf3c817f35354b0f9668d78c1d176ef657062b76836f90dafbe781e551950f3ecd0606f12ff2e053cdb4708f8c9cf0a9030836b6a20ac31632bae248db19bece98b9a098ba86b2c9efcdb4840efc3b7c515ebf7e5784414bfafbbe0e42c3a0dfeed4ce7d6bf7afa062216d2d856840a0da483f407a0f78934933deac4e685ac2a7edaee227a5134193cebb87906e66183536baec0fb75e30aef292292af1b8062cadc3d10c728129a5602401580d85b31833fba37a5a08d78b4f577c04dd448925b3edda3230a5164d3074d7e8643d2c67463d6a2521283d9c90595bf7877bc603fb5778193df6e2087516c55c7de1835f3018dbdde638004c2410ef16ac9b9a25a68ca79db73cb6674e812f5c0a94dc377a4e77fe0747f92340dfda5bf43d3ff911c902099c5c22ff4eec17f3bdb4710231c3ca0e8df8d71d032ba8b92f60c006c6a17c5a163b2075d7bf73e6b5530f38df26e5782a2ab2e3c10e58fcc9ae5c140051f70e77b7117cf7ec86b4e2abcdabb04300c21105622ea78b9959c6a588973965011d458b74acc81cd86af5c2dbd432f9d61a200fd87c970780bec699aae9f3b2c20be8f3d7f512964308ea4d5645ae21586219d06e39cf169d9e781ddec2fb4559954b7b9639eb70d25d351d9443eb7f0ee54281ef9486bce31920c16344e91848b2bc16766b01ff9e6407c24e28ff5d13dc2cfc6d13c2f3fe3c5e467199d2f52ac7aa8907f4b5a6a2435729f60e7f2754b35a31d209a177c3dce024e3c8b08196ead591d2cbf534fe39017bdd305caa25932ea6acc3acd45020350d3b9e7a3d99ad30576ea17f6a38e7813af4a20eb2dd0892ad57656a91b786b321ad44e78d3ceb36f3e5228fc7817e31005a06f203775b8b06250527adbd37cef65f801891554d1292fcc05c696ac15d9c156383c349e6a500d9f657e77306dcf84bb2",
			"e2000800f85f1dd58ba099d598bd545ad1e467135625749ce37b26faca41a2978c8b2b5774de6574741e5f69f3cc00895ca46cd564ac93bdc278b77fa18cc4c66572f853be5868555421045b32cb1d1e4d8260492ca856c5d1c984b1eae0f8217d4f37a6b2458f77c535d63364deb70a056e888577becefd26fe68301ddb604be6b305e6d6232660562efa93b367292b6304974bf5d4fdfd87a5fccc0e924bc7f82493d23669c8528d2a4a30b3274be34dd0935d4e19da24c92eb74ebcdbac7dfe913f0f3d3bd79d5a3157b2dcd39b273f8e2a0dc730d6d82bf33ec98f0dec97c64440d710f2fd34f685ac7fe3aa4e96f86b997cf0c1884f257c72a98415a606797ab17b24246f122e60ecf438165d63094671558b47d58c2c4229960cbffc41896c72d52345a26324f7f3f80e7eb3217f9ea9352cf480cb00a6ff261e48f6b25c2b8667e8e05b5a565a213a68d2ced44ed0662181f068b58e4e008f1dbffc89dcc4a6bf3c817f35354b0f9668d78c1d176ef657062b76836f90dafbe781e551950f3ecd0606f12ff2e053cdb4708f8c9cf0a9030836b6a20ac31632bae248db19bece98b9a098ba86b2c9efcdb4840efc3b7c515ebf7e5784414bfafbbe0e42c3a0dfeed4ce7d6bf7afa062216d2d856840a0da483f407a0f78934933deac4e685ac2a7edaee227a5134193cebb87906e66183536baec0fb75e30aef292292af1b8062cadc3d10c728129a5602401580d85b31833fba37a5a08d78b4f577c04dd448925b3edda3230a5164d3074d7e8643d2c67463d6a2521283d9c90595bf7877bc603fb5778193df6e2087516c55c7de1835f3018dbdde638004c2410ef16ac9b9a25a68ca79db73cb6674e812f5c0a94dc377a4e77fe0747f92340dfda5bf43d3ff911c902099c5c22ff4eec17f3bdb4710231c3ca0e8df8d71d032ba8b92f60c006c6a17c5a163b2075d7bf73e6b5530f38df26e5782a2ab2e3c10e58fcc9ae5c140051f70e77b7117cf7ec86b4e2abcdabb04300c21105622ea78b9959c6a588973965011d458b74acc81cd86af5c2dbd432f9d61a200fd87c970780bec699aae9f3b2c20be8f3d7f512964308ea4d5645ae21586219d06e39cf169d9e781ddec2fb4559954b7b9639eb70d25d351d9443eb7f0ee54281ef9486bce31920c16344e91848b2bc16766b01ff9e6407c24e28ff5d13dc2cfc6d13c2f3fe3c5e467199d2f52ac7aa8907f4b5a6a2435729f60e7f2754b35a31d209a177c3dce024e3c8b08196ead591d2cbf534fe39017bdd305caa25932ea6acc3acd45020350d3b9e7a3d99ad30576ea17f6a38e7813af4a20eb2dd0892ad57656a91b786b321ad44e78d3ceb36f3e5228fc7817e31005a06f203775b8b06250527adbd37cef65f801891554d1292fcc05c696ac15d9c156383c349e6a500d9f657e77306dcf84bb2f85f1dd58ba099d598bd545ad1e467135625749ce37b26faca41a2978c8b2b5774de6574741e5f69f3cc00895ca46cd564ac93bdc278b77fa18cc4c66572f853be5868555421045b32cb1d1e4d8260492ca856c5d1c984b1eae0f8217d4f37a6b2458f77c535d63364deb70a056e888577becefd26fe68301ddb604be6b305e6d6232660562efa93b367292b6304974bf5d4fdfd87a5fccc0e924bc7f82493d23669c8528d2a4a30b3274be34dd0935d4e19da24c92eb74ebcdbac7dfe913f0f3d3bd79d5a3157b2dcd39b273f8e2a0dc730d6d82bf33ec98f0dec97c64440d710f2fd34f685ac7fe3aa4e96f86b997cf0c1884f257c72a98415a606797ab17b24246f122e60ecf438165d63094671558b47d58c2c4229960cbffc41896c72d52345a26324f7f3f80e7eb3217f9ea9352cf480cb00a6ff261e48f6b25c2b8667e8e05b5a565a213a68d2ced44ed0662181f068b58e4e008f1dbffc89dcc4a6bf3c817f35354b0f9668d78c1d176ef657062b76836f90dafbe781e551950f3ecd0606f12ff2e053cdb4708f8c9cf0a9030836b6a20ac31632bae248db19bece98b9a098ba86b2c9efcdb4840efc3b7c515ebf7e5784414bfafbbe0e42c3a0dfeed4ce7d6bf7afa062216d2d856840a0da483f407a0f78934933deac4e685ac2a7edaee227a5134193cebb87906e66183536baec0fb75e30aef292292af1b8062cadc3d10c728129a5602401580d85b31833fba37a5a08d78b4f577c04dd448925b3edda3230a5164d3074d7e8643d2c67463d6a2521283d9c90595bf7877bc603fb5778193df6e2087516c55c7de1835f3018dbdde638004c2410ef16ac9b9a25a68ca79db73cb6674e812f5c0a94dc377a4e77fe0747f92340dfda5bf43d3ff911c902099c5c22ff4eec17f3bdb4710231c3ca0e8df8d71d032ba8b92f60c006c6a17c5a163b2075d7bf73e6b5530f38df26e5782a2ab2e3c10e58fcc9ae5c140051f70e77b7117cf7ec86b4e2abcdabb04300c21105622ea78b9959c6a588973965011d458b74acc81cd86af5c2dbd432f9d61a200fd87c970780bec699aae9f3b2c20be8f3d7f512964308ea4d5645ae21586219d06e39cf169d9e781ddec2fb4559954b7b9639eb70d25d351d9443eb7f0ee54281ef9486bce31920c16344e91848b2bc16766b01ff9e6407c24e28ff5d13dc2cfc6d13c2f3fe3c5e467199d2f52ac7aa8907f4b5a6a2435729f60e7f2754b35a31d209a177c3dce024e3c8b08196ead591d2cbf534fe39017bdd305caa25932ea6acc3acd45020350d3b9e7a3d99ad30576ea17f6a38e7813af4a20eb2dd0892ad57656a91b786b321ad44e78d3ceb36f3e5228fc7817e31005a06f203775b8b06250527adbd37cef65f801891554d1292fcc05c696ac15d9c156383c349e6a500d9f657e77306dcf84bb2",
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
		{"EmptyAtom", "FF", List{}, nil},
		{"ErrMediumIntegerNotImplemented", "C0 00", nil, ErrMediumIntegerNotImplemented},
		{"ErrLongIntegerNotImplemented", "E0 00 00 00", nil, ErrLongIntegerNotImplemented},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, _ := hex.DecodeString(strings.ReplaceAll(tc.data, " ", ""))
			if got, err := Decode(in); !reflect.DeepEqual(got, tc.want) || !errors.Is(err, tc.err) {
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
		{"Broken StartList", "F0 C0 00", nil, ErrMediumIntegerNotImplemented},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, _ := hex.DecodeString(strings.ReplaceAll(tc.data, " ", ""))
			if got, err := Decode(in); !reflect.DeepEqual(got, tc.want) || !errors.Is(err, tc.err) {
				t.Errorf("In(%+v) = %+v, %+v; want %+v, %+v", in, got, err, tc.want, tc.err)
			}
		})
	}

}

func TestEqualBytes(t *testing.T) {
	TestCases := []struct {
		name string
		data interface{}
		comp []byte
		want bool
	}{
		{"Equal byte slices", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"Different byte slices", []byte{1, 2, 3}, []byte{4, 5, 6}, false},
		{"Special nil case", []byte{}, []byte{}, true},
		{"Unrelated type", "not bytes", []byte{1, 2, 3}, false},
		{"Nil input", nil, []byte{1, 2, 3}, false},
	}

	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := EqualBytes(tc.data, tc.comp)
			if result != tc.want {
				t.Errorf("EqualBytes(%v, %v) = %v; want %v", tc.data, tc.comp, result, tc.want)
			}
		})
	}
}

func TestEqualToken(t *testing.T) {
	TestCases := []struct {
		name string
		data interface{}
		comp TokenType
		want bool
	}{
		{"Equal TokenType values", StartList, StartList, true},
		{"Different TokenType values", StartList, EndList, false},
		{"Equal byte slice representation", Token(StartList), StartList, true},
		{"Mismatched byte slice", []byte{0}, StartList, false},
		{"Invalid byte slice length", []byte{0xF0, 0}, StartList, false},
		{"Unrelated type", "StartList", StartList, false},
		{"Nil input", nil, StartList, false},
	}

	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EqualToken(tc.data, tc.comp)
			if got != tc.want {
				t.Errorf("EqualToken(%v, %v) = %v; want %v", tc.data, tc.comp, got, tc.want)
			}
		})
	}
}

func TestEqualUInt(t *testing.T) {
	testCases := []struct {
		name string
		data interface{}
		comp uint
		want bool
	}{
		{"Equal uint values", uint(42), 42, true},
		{"Different uint values", uint(42), 0, false},
		{"Not a uint (int type)", int(42), 42, false},
		{"Input is nil", nil, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EqualUInt(tc.data, tc.comp)
			if got != tc.want {
				t.Errorf("EqualUInt(%v, %v) = %v; want %v", tc.data, tc.comp, got, tc.want)
			}
		})
	}
}
