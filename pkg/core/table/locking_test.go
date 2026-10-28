// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
)

// mbrTestSession builds a session with only the token sizes set.
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

func TestSuggestBufferSizes(t *testing.T) {
	tests := []struct {
		name string
		// Session token sizes.
		hostInd, hostAgg uint
		tperInd, tperAgg uint
		// Table granularities.
		mandatoryWrite  uint32
		recommendedRead uint32
		wantRead        uint
		wantWrite       uint
	}{
		{
			name:            "tper smaller than host",
			hostInd:         1992,
			hostAgg:         1992,
			tperInd:         968,
			tperAgg:         968,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        968 - 128,
			wantWrite:       968 - 128,
		},
		{
			// The host buffer only bounds reads.
			name:            "host smaller than tper",
			hostInd:         512,
			hostAgg:         512,
			tperInd:         2048,
			tperAgg:         2048,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        512 - 128,
			wantWrite:       2048 - 128,
		},
		{
			name:            "agg token size ignored",
			hostInd:         1000,
			hostAgg:         2000,
			tperInd:         3000,
			tperAgg:         5000,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        1000 - 128,
			wantWrite:       3000 - 128,
		},
		{
			// Reads ignore the write granularity.
			name:            "mandatory write granularity alignment",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  512,
			recommendedRead: 1,
			wantRead:        1872,
			wantWrite:       1536,
		},
		{
			name:            "recommended access granularity alignment",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  1,
			recommendedRead: 256,
			wantRead:        1792,
			wantWrite:       1792,
		},
		{
			name:            "both granularities applied",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  512,
			recommendedRead: 256,
			wantRead:        1792,
			wantWrite:       1536,
		},
		{
			name:            "non power of two granularities",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  24,
			recommendedRead: 40,
			wantRead:        1840,
			wantWrite:       1800,
		},
		{
			// Writes fall back to the mandatory granularity alone.
			name:            "recommended alignment dropped if too large",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  512,
			recommendedRead: 4000,
			wantRead:        1872,
			wantWrite:       1536,
		},
		{
			name:            "zero granularities",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  0,
			recommendedRead: 0,
			wantRead:        1872,
			wantWrite:       1872,
		},
		{
			name:            "mandatory granularity larger than token",
			hostInd:         2000,
			hostAgg:         2000,
			tperInd:         2000,
			tperAgg:         2000,
			mandatoryWrite:  4096,
			recommendedRead: 1,
			wantRead:        1872,
			wantWrite:       0,
		},
		{
			// Samsung 970 EVO Plus with WithMaxComPacketSize(2048).
			name:            "small host compacket with 4k granularity",
			hostInd:         1992,
			hostAgg:         1992,
			tperInd:         65540,
			tperAgg:         968,
			mandatoryWrite:  4096,
			recommendedRead: 4096,
			wantRead:        1992 - 128,
			wantWrite:       61440,
		},
		{
			name:            "tiny tokens underflow guard",
			hostInd:         100,
			hostAgg:         100,
			tperInd:         100,
			tperAgg:         100,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        0,
			wantWrite:       0,
		},
		{
			name:            "exactly framing reserve",
			hostInd:         128,
			hostAgg:         128,
			tperInd:         128,
			tperAgg:         128,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        0,
			wantWrite:       0,
		},
		{
			name:            "one byte above framing reserve",
			hostInd:         129,
			hostAgg:         129,
			tperInd:         129,
			tperAgg:         129,
			mandatoryWrite:  1,
			recommendedRead: 1,
			wantRead:        1,
			wantWrite:       1,
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
			if got := m.SuggestBufferSize(s); got != tt.wantRead {
				t.Errorf("SuggestBufferSize() = %d, want %d", got, tt.wantRead)
			}
			if got := m.SuggestWriteBufferSize(s); got != tt.wantWrite {
				t.Errorf("SuggestWriteBufferSize() = %d, want %d", got, tt.wantWrite)
			}
		})
	}
}

// TestSuggestBufferSizeSymmetric checks that only the smaller side matters.
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

type mbrWrite struct {
	off  uint32
	data []byte
}

func TestLoadPBAImage(t *testing.T) {
	image := make([]byte, 2500)
	for i := range image {
		image[i] = byte(i%251 + 1)
	}

	tests := []struct {
		name        string
		image       []byte
		chunk       uint
		granularity uint
		size        uint
		// Expected writes as (offset, length).
		want    [][2]uint
		wantErr string
	}{
		{
			name:        "unaligned tail without granularity",
			image:       image,
			chunk:       1000,
			granularity: 1,
			size:        1 << 20,
			want:        [][2]uint{{0, 1000}, {1000, 1000}, {2000, 500}},
		},
		{
			name:        "tail padded to granularity",
			image:       image,
			chunk:       1024,
			granularity: 512,
			size:        1 << 20,
			want:        [][2]uint{{0, 1024}, {1024, 1024}, {2048, 512}},
		},
		{
			name:        "aligned image not padded",
			image:       image[:2048],
			chunk:       1024,
			granularity: 512,
			size:        2048,
			want:        [][2]uint{{0, 1024}, {1024, 1024}},
		},
		{
			name:        "image larger than shadow MBR",
			image:       image,
			chunk:       1000,
			granularity: 1,
			size:        2000,
			wantErr:     "larger than the shadow MBR",
		},
		{
			name:        "padding larger than shadow MBR",
			image:       image,
			chunk:       1024,
			granularity: 512,
			size:        2500,
			wantErr:     "larger than the shadow MBR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var writes []mbrWrite
			err := loadPBAImage(tt.image, tt.chunk, tt.granularity, tt.size, func(p []byte, off uint32) (int, error) {
				writes = append(writes, mbrWrite{off, append([]byte(nil), p...)})
				return len(p), nil
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadPBAImage() error = %v, want one containing %q", err, tt.wantErr)
				}
				if len(writes) != 0 {
					t.Errorf("loadPBAImage() wrote %d chunks before failing, want 0", len(writes))
				}
				return
			}
			if err != nil {
				t.Fatalf("loadPBAImage() error: %v", err)
			}
			if len(writes) != len(tt.want) {
				t.Fatalf("loadPBAImage() made %d writes, want %d", len(writes), len(tt.want))
			}
			for i, w := range writes {
				off, l := tt.want[i][0], tt.want[i][1]
				if uint(w.off) != off || uint(len(w.data)) != l {
					t.Errorf("write %d = (%d, %d bytes), want (%d, %d bytes)", i, w.off, len(w.data), off, l)
					continue
				}
				for j, b := range w.data {
					var want byte
					if pos := off + uint(j); pos < uint(len(tt.image)) {
						want = tt.image[pos]
					}
					if b != want {
						t.Errorf("write %d byte %d = %#x, want %#x", i, j, b, want)
						break
					}
				}
			}
		})
	}
}

func TestLoadPBAImageWriteError(t *testing.T) {
	calls := 0
	err := loadPBAImage(make([]byte, 3000), 1000, 1, 1<<20, func(p []byte, off uint32) (int, error) {
		calls++
		if off == 1000 {
			return 0, errors.New("method returned status FAIL")
		}
		return len(p), nil
	})
	if err == nil || !strings.Contains(err.Error(), "offset 1000") {
		t.Fatalf("loadPBAImage() error = %v, want one mentioning offset 1000", err)
	}
	if calls != 2 {
		t.Errorf("loadPBAImage() made %d writes, want 2 (stop at the first error)", calls)
	}
}
