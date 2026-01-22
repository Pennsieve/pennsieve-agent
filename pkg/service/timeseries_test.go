package service

import (
	"testing"
)

func TestCalculateCropIndices(t *testing.T) {
	tests := []struct {
		name          string
		blockStart    uint64
		blockEnd      uint64
		requestStart  uint64
		requestEnd    uint64
		rate          float64
		wantStartIdx  int64
		wantEndIdx    int64
	}{
		{
			name:         "No cropping needed - block fully within request",
			blockStart:   1000000,  // 1 second
			blockEnd:     2000000,  // 2 seconds
			requestStart: 500000,   // 0.5 seconds
			requestEnd:   2500000,  // 2.5 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 0,
			wantEndIdx:   -1,
		},
		{
			name:         "Crop from beginning only",
			blockStart:   1000000,  // 1 second
			blockEnd:     3000000,  // 3 seconds
			requestStart: 1500000,  // 1.5 seconds
			requestEnd:   4000000,  // 4 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 4000,     // 0.5 seconds * 1000 Hz * 8 bytes = 4000
			wantEndIdx:   -1,
		},
		{
			name:         "Crop from end only",
			blockStart:   1000000,  // 1 second
			blockEnd:     3000000,  // 3 seconds
			requestStart: 500000,   // 0.5 seconds
			requestEnd:   2500000,  // 2.5 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 0,
			wantEndIdx:   12000,    // 1.5 seconds * 1000 Hz * 8 bytes = 12000
		},
		{
			name:         "Crop from both beginning and end",
			blockStart:   1000000,  // 1 second
			blockEnd:     4000000,  // 4 seconds
			requestStart: 1500000,  // 1.5 seconds
			requestEnd:   3500000,  // 3.5 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 4000,     // 0.5 seconds * 1000 Hz * 8 bytes = 4000
			wantEndIdx:   20000,    // 2.5 seconds * 1000 Hz * 8 bytes = 20000
		},
		{
			name:         "Non-integer start time with fractional seconds",
			blockStart:   1000000,  // 1 second
			blockEnd:     3000000,  // 3 seconds
			requestStart: 1250000,  // 1.25 seconds
			requestEnd:   3000000,  // 3 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 2000,     // 0.25 seconds * 1000 Hz * 8 bytes = 2000
			wantEndIdx:   -1,
		},
		{
			name:         "Non-integer end time with fractional seconds",
			blockStart:   1000000,  // 1 second
			blockEnd:     3000000,  // 3 seconds
			requestStart: 1000000,  // 1 second
			requestEnd:   2750000,  // 2.75 seconds
			rate:         1000.0,   // 1000 Hz
			wantStartIdx: 0,
			wantEndIdx:   14000,    // 1.75 seconds * 1000 Hz * 8 bytes = 14000
		},
		{
			name:         "High sample rate with non-integer times",
			blockStart:   1000000,  // 1 second
			blockEnd:     2000000,  // 2 seconds
			requestStart: 1100000,  // 1.1 seconds
			requestEnd:   1900000,  // 1.9 seconds
			rate:         5000.0,   // 5000 Hz
			wantStartIdx: 4000,     // 0.1 seconds * 5000 Hz * 8 bytes = 4000
			wantEndIdx:   36000,    // 0.9 seconds * 5000 Hz * 8 bytes = 36000
		},
		{
			name:         "Low sample rate with non-integer times",
			blockStart:   1000000,  // 1 second
			blockEnd:     10000000, // 10 seconds
			requestStart: 2500000,  // 2.5 seconds
			requestEnd:   7500000,  // 7.5 seconds
			rate:         100.0,    // 100 Hz
			wantStartIdx: 1200,     // 1.5 seconds * 100 Hz * 8 bytes = 1200
			wantEndIdx:   5200,     // 6.5 seconds * 100 Hz * 8 bytes = 5200
		},
		{
			name:         "Very small time differences",
			blockStart:   1000000,  // 1 second
			blockEnd:     1001000,  // 1.001 seconds
			requestStart: 1000100,  // 1.0001 seconds
			requestEnd:   1000900,  // 1.0009 seconds
			rate:         10000.0,  // 10 kHz
			wantStartIdx: 8,        // 0.0001 seconds * 10000 Hz * 8 bytes = 8
			wantEndIdx:   72,       // 0.0009 seconds * 10000 Hz * 8 bytes = 72
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStartIdx, gotEndIdx := calculateCropIndices(
				tt.blockStart,
				tt.blockEnd,
				tt.requestStart,
				tt.requestEnd,
				tt.rate,
			)
			
			if gotStartIdx != tt.wantStartIdx {
				t.Errorf("calculateCropIndices() startIdx = %v, want %v", gotStartIdx, tt.wantStartIdx)
			}
			if gotEndIdx != tt.wantEndIdx {
				t.Errorf("calculateCropIndices() endIdx = %v, want %v", gotEndIdx, tt.wantEndIdx)
			}
		})
	}
}

func TestCalculateCropIndices_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		blockStart    uint64
		blockEnd      uint64
		requestStart  uint64
		requestEnd    uint64
		rate          float64
		wantStartIdx  int64
		wantEndIdx    int64
	}{
		{
			name:         "Request exactly matches block",
			blockStart:   1000000,
			blockEnd:     2000000,
			requestStart: 1000000,
			requestEnd:   2000000,
			rate:         1000.0,
			wantStartIdx: 0,
			wantEndIdx:   -1,
		},
		{
			name:         "Request starts at block start but ends before block end",
			blockStart:   1000000,
			blockEnd:     3000000,
			requestStart: 1000000,
			requestEnd:   2000000,
			rate:         1000.0,
			wantStartIdx: 0,
			wantEndIdx:   8000, // 1 second * 1000 Hz * 8 bytes
		},
		{
			name:         "Request starts after block start but ends at block end",
			blockStart:   1000000,
			blockEnd:     3000000,
			requestStart: 2000000,
			requestEnd:   3000000,
			rate:         1000.0,
			wantStartIdx: 8000, // 1 second * 1000 Hz * 8 bytes
			wantEndIdx:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStartIdx, gotEndIdx := calculateCropIndices(
				tt.blockStart,
				tt.blockEnd,
				tt.requestStart,
				tt.requestEnd,
				tt.rate,
			)
			
			if gotStartIdx != tt.wantStartIdx {
				t.Errorf("calculateCropIndices() startIdx = %v, want %v", gotStartIdx, tt.wantStartIdx)
			}
			if gotEndIdx != tt.wantEndIdx {
				t.Errorf("calculateCropIndices() endIdx = %v, want %v", gotEndIdx, tt.wantEndIdx)
			}
		})
	}
}