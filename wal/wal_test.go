package wal_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cockroachdb/pebble/internal/base"
	"github.com/cockroachdb/pebble/wal"
	"github.com/stretchr/testify/require"
)

func TestNumWALString(t *testing.T) {
	tests := []struct {
		num      wal.NumWAL
		expected string
	}{
		{0, "000000"},
		{1, "000001"},
		{999999, "999999"},
		{wal.NumWAL(base.DiskFileNum(1)), "000001"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("num=%d", tc.num), func(t *testing.T) {
			require.Equal(t, tc.expected, tc.num.String())
		})
	}
}

func TestLogNameIndexString(t *testing.T) {
	tests := []struct {
		index    wal.LogNameIndex
		expected string
	}{
		{0, "000"},
		{1, "001"},
		{999, "999"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("index=%d", tc.index), func(t *testing.T) {
			require.Equal(t, tc.expected, tc.index.String())
		})
	}
}

func TestMakeAndParseLogFilename(t *testing.T) {
	tests := []struct {
		num      wal.NumWAL
		index    wal.LogNameIndex
		expected string
		valid    bool
	}{
		{0, 0, "000000.log", true},
		{1, 0, "000001.log", true},
		{1, 1, "000001-001.log", true},
		{999999, 999, "999999-999.log", true},
		{wal.NumWAL(base.DiskFileNum(1)), 0, "000001.log", true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("num=%d,index=%d", tc.num, tc.index), func(t *testing.T) {
			filename := fmt.Sprintf("%s-%s.log", tc.num.String(), tc.index.String())
			if tc.index == 0 {
				filename = fmt.Sprintf("%s.log", tc.num.String())
			}
			require.Equal(t, tc.expected, filename)

			num, index, ok := wal.ParseLogFilename(filename)
			require.Equal(t, tc.valid, ok)
			if tc.valid {
				require.Equal(t, tc.num, num)
				require.Equal(t, tc.index, index)
			}
		})
	}

	// Test invalid filenames
	invalidTests := []struct {
		filename string
	}{
		{"invalid.log"},
		{"000001.txt"},
		{"000001-.log"},
		{"-001.log"},
		{"000001-abc.log"},
		{""},
		{".log"},
		{"000001-"},
		{"abcdef.log"},
	}

	for _, tc := range invalidTests {
		t.Run(fmt.Sprintf("invalid=%s", tc.filename), func(t *testing.T) {
			_, _, ok := wal.ParseLogFilename(tc.filename)
			require.False(t, ok)
		})
	}
}

func TestOptionsDirs(t *testing.T) {
	tests := []struct {
		name     string
		opts     wal.Options
		expected int
	}{
		{
			name: "primary only",
			opts: wal.Options{
				Primary: wal.Dir{Dirname: "/primary"},
			},
			expected: 1,
		},
		{
			name: "primary and secondary",
			opts: wal.Options{
				Primary:   wal.Dir{Dirname: "/primary"},
				Secondary: wal.Dir{Dirname: "/secondary"},
			},
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dirs := tc.opts.Dirs()
			require.Len(t, dirs, tc.expected)
			require.Equal(t, tc.opts.Primary, dirs[0])
			if tc.expected > 1 {
				require.Equal(t, tc.opts.Secondary, dirs[1])
			}
		})
	}
}

func TestFailoverOptionsEnsureDefaults(t *testing.T) {
	tests := []struct {
		name string
		opts wal.FailoverOptions
	}{
		{
			name: "empty options",
			opts: wal.FailoverOptions{},
		},
		{
			name: "partial options",
			opts: wal.FailoverOptions{
				PrimaryDirProbeInterval: 2 * time.Second,
			},
		},
		{
			name: "custom unhealthy threshold",
			opts: wal.FailoverOptions{
				UnhealthyOperationLatencyThreshold: func() (time.Duration, bool) {
					return 200 * time.Millisecond, true
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts.EnsureDefaults()
			require.NotZero(t, tc.opts.PrimaryDirProbeInterval)
			require.NotZero(t, tc.opts.HealthyProbeLatencyThreshold)
			require.NotZero(t, tc.opts.HealthyInterval)
			require.NotZero(t, tc.opts.UnhealthySamplingInterval)
			require.NotNil(t, tc.opts.UnhealthyOperationLatencyThreshold)
			require.NotZero(t, tc.opts.ElevatedWriteStallThresholdLag)

			// Verify default values
			if tc.name == "empty options" {
				require.Equal(t, time.Second, tc.opts.PrimaryDirProbeInterval)
				require.Equal(t, 25*time.Millisecond, tc.opts.HealthyProbeLatencyThreshold)
				require.Equal(t, 15*time.Second, tc.opts.HealthyInterval)
				require.Equal(t, 100*time.Millisecond, tc.opts.UnhealthySamplingInterval)
				require.Equal(t, 60*time.Second, tc.opts.ElevatedWriteStallThresholdLag)
			}
		})
	}
}

func TestOffsetString(t *testing.T) {
	tests := []struct {
		name     string
		offset   wal.Offset
		expected string
	}{
		{
			name: "basic",
			offset: wal.Offset{
				PhysicalFile: "test.log",
				Physical:     100,
			},
			expected: "(test.log: 100)",
		},
		{
			name: "with previous files",
			offset: wal.Offset{
				PhysicalFile:        "test.log",
				Physical:            100,
				PreviousFilesBytes: 1000,
			},
			expected: "(test.log: 100), 1000 from previous files",
		},
		{
			name: "zero values",
			offset: wal.Offset{
				PhysicalFile: "",
				Physical:     0,
			},
			expected: "(: 0)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.offset.String())
		})
	}
}
