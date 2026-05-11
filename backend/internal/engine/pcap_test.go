package engine

import (
	"testing"
)

func TestHtons(t *testing.T) {
	tests := []struct {
		input    uint16
		expected uint16
	}{
		{0x1234, 0x3412},
		{0x0001, 0x0100},
		{0x00ff, 0xff00},
		{0x0000, 0x0000},
		{0xffff, 0xffff},
	}

	for _, tt := range tests {
		got := htons(tt.input)
		if got != tt.expected {
			t.Errorf("htons(0x%04x) = 0x%04x, want 0x%04x", tt.input, got, tt.expected)
		}
	}
}

func TestHtonsRoundtrip(t *testing.T) {
	tests := []uint16{0x1234, 0xabcd, 0x0800, 0x0001, 0xffff}
	for _, v := range tests {
		if htons(htons(v)) != v {
			t.Errorf("htons(htons(0x%04x)) != 0x%04x", v, v)
		}
	}
}

func TestReadPCAPFile_NotImplemented(t *testing.T) {
	sender := &PCAPSender{}
	_, err := sender.ReadPCAPFile("test.pcap")
	if err == nil {
		t.Error("expected error for unimplemented ReadPCAPFile, got nil")
	}
}
