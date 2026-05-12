package engine

import (
	"encoding/binary"
	"testing"

	"dns-sender/pkg/models"
)

func TestBuildDNSQuery(t *testing.T) {
	query, err := BuildDNSQuery("example.com", 0x1234)
	if err != nil {
		t.Fatalf("BuildDNSQuery failed: %v", err)
	}

	if len(query) < 16 {
		t.Fatalf("DNS query too short: got %d bytes", len(query))
	}

	if query[0] != 0x12 || query[1] != 0x34 {
		t.Errorf("Transaction ID mismatch: expected 0x1234, got 0x%02x%02x", query[0], query[1])
	}

	if query[2] != 0x01 || query[3] != 0x00 {
		t.Errorf("Expected flags 0x0100, got 0x%02x%02x", query[2], query[3])
	}
}

func TestParseDomain(t *testing.T) {
	tests := []struct {
		domain  string
		wantLen int
	}{
		{"example.com", 13},  // 7example3com0
		{"test.example.org", 18},  // 4test7example3org0
		{"localhost", 11},  // 9localhost0
	}

	for _, tt := range tests {
		result, err := parseDomain(tt.domain)
		if err != nil {
			t.Errorf("parseDomain(%q) unexpected error: %v", tt.domain, err)
			continue
		}
		if len(result) != tt.wantLen {
			t.Errorf("parseDomain(%q) length = %d, want %d", tt.domain, len(result), tt.wantLen)
		}
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"example.com"`, "example.com"},
		{`"test"`, "test"},
		{`noquotes`, "noquotes"},
		{`"a"`, "a"},
		{``, ""},
	}

	for _, tt := range tests {
		result := trimQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("trimQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseCSVDomains(t *testing.T) {
	content := `example.com
google.com
"quoted.domain.com"
ns1.example.org
`

	domains := ParseCSVDomains(content)

	if len(domains) != 4 {
		t.Fatalf("Expected 4 domains, got %d", len(domains))
	}

	if domains[0] != "example.com" {
		t.Errorf("First domain = %q, want %q", domains[0], "example.com")
	}
	if domains[1] != "google.com" {
		t.Errorf("Second domain = %q, want %q", domains[1], "google.com")
	}
	if domains[2] != "quoted.domain.com" {
		t.Errorf("Third domain = %q, want %q", domains[2], "quoted.domain.com")
	}
	if domains[3] != "ns1.example.org" {
		t.Errorf("Fourth domain = %q, want %q", domains[3], "ns1.example.org")
	}
}

func TestParseCSVDomainsWithCRLF(t *testing.T) {
	content := "example.com\r\ngoogle.com\r\n"

	domains := ParseCSVDomains(content)

	if len(domains) != 2 {
		t.Fatalf("Expected 2 domains, got %d", len(domains))
	}
}

func TestParseCSVDomainsEmpty(t *testing.T) {
	tests := []struct {
		content string
		want    int
	}{
		{"", 0},
		{"\n", 0},
		{"\r\n", 0},
	}

	for _, tt := range tests {
		domains := ParseCSVDomains(tt.content)
		if len(domains) != tt.want {
			t.Errorf("ParseCSVDomains(%q) returned %d domains, want %d", tt.content, len(domains), tt.want)
		}
	}
}

func TestParseCSVDomainsWithEscapedQuotes(t *testing.T) {
	content := `"domain""with""quotes".com
"hello ""world"""
normal.com
`
	domains := ParseCSVDomains(content)

	if len(domains) != 3 {
		t.Fatalf("Expected 3 domains, got %d", len(domains))
	}
	if domains[0] != `domain"with"quotes.com` {
		t.Errorf("First domain = %q", domains[0])
	}
	if domains[1] != `hello "world"` {
		t.Errorf("Second domain = %q", domains[1])
	}
	if domains[2] != "normal.com" {
		t.Errorf("Third domain = %q", domains[2])
	}
}

func TestParseDomainInvalidLabelLength(t *testing.T) {
	longLabel := make([]byte, 64)
	for i := range longLabel {
		longLabel[i] = 'a'
	}
	_, err := parseDomain(string(longLabel))
	if err == nil {
		t.Error("expected error for 64-char label, got nil")
	}
}

func TestParseDomainInvalidTotalLength(t *testing.T) {
	// 64 labels × 4 chars each = 256 chars (exceeds 253)
	domain := ""
	for i := 0; i < 64; i++ {
		if i > 0 {
			domain += "."
		}
		domain += "abcd"
	}
	_, err := parseDomain(domain)
	if err == nil {
		t.Error("expected error for domain exceeding 253 chars, got nil")
	}
}

func TestNewQoSControllerDefaults(t *testing.T) {
	cfg := models.QoSConfig{
		TargetQPS:  100,
		Jitter:     0.1,
		DelayMinMs: 10,
		DelayMaxMs: 50,
	}

	qos := NewQoSController(cfg)

	if qos.targetQPS != 100 {
		t.Errorf("targetQPS = %d, want 100", qos.targetQPS)
	}
	if qos.jitter != 0.1 {
		t.Errorf("jitter = %f, want 0.1", qos.jitter)
	}
}

func TestNewQoSControllerZeroQPS(t *testing.T) {
	cfg := models.QoSConfig{}

	qos := NewQoSController(cfg)

	if qos.targetQPS != 100 {
		t.Errorf("targetQPS = %d, want default 100", qos.targetQPS)
	}
}

func TestNewQoSControllerBatchSize(t *testing.T) {
	tests := []struct {
		qps       int
		wantBatch int
	}{
		{100, 1},
		{1000, 1},
		{2000, 2},  // interval=0.5ms, batch=2
		{10000, 10},
		{20000, 20},
		{100000, 100},
	}

	for _, tt := range tests {
		cfg := models.QoSConfig{TargetQPS: tt.qps}
		qos := NewQoSController(cfg)
		if qos.BatchSize() != tt.wantBatch {
			t.Errorf("QPS=%d: BatchSize()=%d, want %d", tt.qps, qos.BatchSize(), tt.wantBatch)
		}
	}
}

func TestChecksum(t *testing.T) {
	header := make([]byte, 20)
	header[0] = 0x45
	header[1] = 0x00

	sum := checksum(header)

	if sum == 0 {
		t.Error("checksum returned 0, which is unusual for valid header")
	}
}

func TestChecksumConsistency(t *testing.T) {
	data := []byte{0x45, 0x00, 0x00, 0x1c, 0x40, 0x00, 0x40, 0x06}

	sum1 := checksum(data)
	sum2 := checksum(data)

	if sum1 != sum2 {
		t.Errorf("checksum not consistent: got %d and %d", sum1, sum2)
	}
}

func TestParseMAC(t *testing.T) {
	tests := []struct {
		input    string
		wantErr  bool
	}{
		{"aa:bb:cc:dd:ee:ff", false},
		{"11:22:33:44:55:66", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		_, err := ParseMAC(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMAC(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestUDPChecksum(t *testing.T) {
	payload := []byte{0x00, 0x01, 0x02, 0x03}
	sum := udpChecksum("192.168.1.1", "192.168.1.2", payload)
	// A proper checksum over real data should never be zero
	if sum == 0 {
		t.Error("udpChecksum returned 0, expected non-zero checksum")
	}
}

func TestUDPChecksumConsistency(t *testing.T) {
	data := []byte{0x00, 0x35, 0x00, 0x35, 0x00, 0x20, 0x00, 0x00}
	data = append(data, make([]byte, 24)...)

	sum1 := udpChecksum("10.0.0.1", "10.0.0.2", data)
	sum2 := udpChecksum("10.0.0.1", "10.0.0.2", data)
	if sum1 != sum2 {
		t.Errorf("udpChecksum not consistent: %d vs %d", sum1, sum2)
	}
	if sum1 == 0 {
		t.Error("udpChecksum returned 0 for real DNS-like data")
	}
}

func TestUDPChecksumInvalidIP(t *testing.T) {
	sum := udpChecksum("bad", "192.168.1.1", []byte{1, 2, 3})
	if sum != 0 {
		t.Errorf("udpChecksum with invalid IP: got %d, want 0", sum)
	}
}

func TestBuildUDPPacketChecksumNonZero(t *testing.T) {
	payload := []byte{
		0x12, 0x34, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x78, 0x61,
		0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d,
		0x00, 0x00, 0x01, 0x00, 0x01,
	}
	pkt := BuildUDPPacket(1234, 53, "192.168.1.1", "8.8.8.8", payload)
	// checksum is at bytes 6-7 of UDP header
	cs := binary.BigEndian.Uint16(pkt[6:8])
	if cs == 0 {
		t.Error("BuildUDPPacket produced zero checksum")
	}
}