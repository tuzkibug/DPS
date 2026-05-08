package engine

import (
	"testing"

	"dns-sender/pkg/models"
)

func TestBuildDNSQuery(t *testing.T) {
	query := BuildDNSQuery("example.com", 0x1234)

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
		result := parseDomain(tt.domain)
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