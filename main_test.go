package main

import (
	"os"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{2048, "2.0 KB"},
		{3145728, "3.0 MB"},
	}

	for _, test := range tests {
		result := formatBytes(test.input)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestGetMemoryUsage(t *testing.T) {
	// Test with current process PID
	pid := os.Getpid()
	memory := getMemoryUsage(pid)
	
	// Memory usage should be greater than 0 for a running process
	if memory == 0 {
		t.Errorf("getMemoryUsage(%d) returned 0, expected > 0", pid)
	}
	
	// Test with invalid PID
	invalidPid := 999999
	memory = getMemoryUsage(invalidPid)
	// Invalid PID should return 0
	if memory != 0 {
		t.Errorf("getMemoryUsage(%d) returned %d, expected 0 for invalid PID", invalidPid, memory)
	}
}