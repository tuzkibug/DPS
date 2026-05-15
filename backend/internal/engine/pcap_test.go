package engine

import (
	"testing"
)

func TestRandomIPv4(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ip := randomIPv4()
		if ip == nil {
			t.Fatal("randomIPv4 returned nil")
		}
		if ip.To4() == nil {
			t.Fatal("randomIPv4 did not return a valid IPv4 address")
		}
		seen[ip.String()] = true
	}
	if len(seen) < 10 {
		t.Errorf("randomIPv4 generated only %d unique IPs, expected at least 10", len(seen))
	}
}

func TestRandomMAC(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		mac := randomMAC()
		if mac == nil {
			t.Fatal("randomMAC returned nil")
		}
		if len(mac) != 6 {
			t.Fatalf("randomMAC returned %d bytes, want 6", len(mac))
		}
		// Verify unicast: bit 0 of first byte must be 0
		if mac[0]&0x01 != 0 {
			t.Errorf("randomMAC first byte 0x%02x has multicast bit set", mac[0])
		}
		// Verify locally administered: bit 1 of first byte must be 1
		if mac[0]&0x02 == 0 {
			t.Errorf("randomMAC first byte 0x%02x has locally-administered bit clear", mac[0])
		}
		seen[mac.String()] = true
	}
	if len(seen) < 10 {
		t.Errorf("randomMAC generated only %d unique MACs, expected at least 10", len(seen))
	}
}

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
