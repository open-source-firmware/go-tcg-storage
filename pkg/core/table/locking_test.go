// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"testing"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
)

// mbrTestSession builds a minimal *core.Session carrying only the Host/TPer
// token sizes that SuggestBufferSize inspects.
func mbrTestSession(hostInd, hostAgg, tperInd, tperAgg uint) *core.Session {
	cs := &core.ControlSession{
		HostProperties: core.HostProperties{
			MaxIndTokenSize: hostInd,
			MaxAggTokenSize: hostAgg,
		},
		TPerProperties: core.TPerProperties{
			MaxIndTokenSize: tperInd,
			MaxAggTokenSize: tperAgg,
		},
	}
	return &core.Session{ControlSession: cs}
}

func TestSuggestBufferSize(t *testing.T) {
	tests := []struct {
		name string
		// Session token sizes.
		hostInd, hostAgg uint
		tperInd, tperAgg uint
		// Table granularities.
		mandatoryWrite  uint32
		recommendedRead uint32
		want            uint
	}{
		{
			// Common case: the drive advertises a smaller token than the host,
			// so the TPer bounds the transfer (matches the previous behaviour).
			name:            "tper smaller than host",
			hostInd:         1992,
			hostAgg:         1992,
			tperInd:         968,
			tperAgg:         968,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            968 - 128,
		},
		{
			// Regression: host receive buffer is the smaller side. A read sized
			// to the TPer would overflow the host, so the host must bound it.
			name:            "host smaller than tper",
			hostInd:         512,
			hostAgg:         512,
			tperInd:         2048,
			tperAgg:         2048,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            512 - 128,
		},
		{
			name:            "equal sizes",
			hostInd:         1024,
			hostAgg:         1024,
			tperInd:         1024,
			tperAgg:         1024,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            1024 - 128,
		},
		{
			// The larger of Ind/Agg is picked per side before taking the min:
			// host max = 2000 (Agg), tper max = 3000 (Ind) -> min = 2000.
			name:            "agg token larger than ind per side",
			hostInd:         100,
			hostAgg:         2000,
			tperInd:         3000,
			tperAgg:         500,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            2000 - 128,
		},
		{
			// min = 1872; align down to MandatoryWriteGranularity 512 -> 1536.
			name:            "mandatory write granularity alignment",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  512,
			recommendedRead: 1,
			want:            1536,
		},
		{
			// min = 1872; align down to RecommendedAccessGranularity 256 -> 1792.
			name:            "recommended access granularity alignment",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  1,
			recommendedRead: 256,
			want:            1792,
		},
		{
			// Both granularities applied in sequence: 1872 -> 1536 -> 1536.
			name:            "both granularities applied",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  512,
			recommendedRead: 256,
			want:            1536,
		},
		{
			// Underflow guard: min <= 128 must return 0, never wrap around.
			name:            "tiny tokens underflow guard",
			hostInd:         100,
			hostAgg:         100,
			tperInd:         100,
			tperAgg:         100,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            0,
		},
		{
			// Boundary: exactly 128 leaves no room for framing -> 0.
			name:            "exactly framing reserve",
			hostInd:         128,
			hostAgg:         128,
			tperInd:         128,
			tperAgg:         128,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            0,
		},
		{
			// Boundary: one byte above the reserve yields one usable byte.
			name:            "one byte above framing reserve",
			hostInd:         129,
			hostAgg:         129,
			tperInd:         129,
			tperAgg:         129,
			mandatoryWrite:  1,
			recommendedRead: 1,
			want:            1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MBRTableInfo{
				Size:                         1 << 20,
				MandatoryWriteGranularity:    tt.mandatoryWrite,
				RecommendedAccessGranularity: tt.recommendedRead,
			}
			s := mbrTestSession(tt.hostInd, tt.hostAgg, tt.tperInd, tt.tperAgg)
			if got := m.SuggestBufferSize(s); got != tt.want {
				t.Errorf("SuggestBufferSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestSuggestBufferSizeSymmetric documents the intended invariant: the result
// depends only on the smaller of the two sides, so swapping which side is
// smaller yields the same size for a given minimum.
func TestSuggestBufferSizeSymmetric(t *testing.T) {
	m := &MBRTableInfo{
		Size:                         1 << 20,
		MandatoryWriteGranularity:    1,
		RecommendedAccessGranularity: 1,
	}

	hostSmaller := m.SuggestBufferSize(mbrTestSession(1000, 1000, 4000, 4000))
	tperSmaller := m.SuggestBufferSize(mbrTestSession(4000, 4000, 1000, 1000))

	if hostSmaller != tperSmaller {
		t.Errorf("expected symmetric result, host-smaller=%d tper-smaller=%d",
			hostSmaller, tperSmaller)
	}
	if want := uint(1000 - 128); hostSmaller != want {
		t.Errorf("SuggestBufferSize() = %d, want %d", hostSmaller, want)
	}
}
