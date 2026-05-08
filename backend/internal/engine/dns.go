package engine

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"time"

	"dns-sender/pkg/models"
)

type DNSPacket struct {
	TransactionID uint16
	Domain       string
	QType        uint16
}

func BuildDNSQuery(domain string, txID uint16) []byte {
	header := make([]byte, 12)

	binary.BigEndian.PutUint16(header[0:2], txID)
	binary.BigEndian.PutUint16(header[2:4], 0x0100)
	binary.BigEndian.PutUint16(header[4:6], 1)
	binary.BigEndian.PutUint16(header[6:8], 0)
	binary.BigEndian.PutUint16(header[8:10], 0)
	binary.BigEndian.PutUint16(header[10:12], 0)

	labels := parseDomain(domain)
	query := append(header, labels...)

	tail := make([]byte, 4)
	binary.BigEndian.PutUint16(tail[0:2], 1) // QTYPE A
	binary.BigEndian.PutUint16(tail[2:4], 1) // QCLASS IN
	query = append(query, tail...)

	return query
}

func parseDomain(domain string) []byte {
	var result []byte
	labels := []byte{}
	for _, c := range []byte(domain) {
		if c == '.' {
			result = append(result, byte(len(labels)))
			result = append(result, labels...)
			labels = []byte{}
		} else {
			labels = append(labels, c)
		}
	}
	if len(labels) > 0 {
		result = append(result, byte(len(labels)))
		result = append(result, labels...)
	}
	result = append(result, 0)
	return result
}

func ParseCSVDomains(content string) []string {
	var domains []string
	lines := []byte{}
	inQuote := false

	for _, c := range []byte(content) {
		if c == '"' {
			inQuote = !inQuote
		} else if c == '\n' && !inQuote {
			if len(lines) > 0 {
				domain := string(lines)
				domain = trimQuotes(domain)
				if len(domain) > 0 {
					domains = append(domains, domain)
				}
			}
			lines = []byte{}
		} else if c != '\r' {
			lines = append(lines, c)
		}
	}

	if len(lines) > 0 {
		domain := string(lines)
		domain = trimQuotes(domain)
		if len(domain) > 0 {
			domains = append(domains, domain)
		}
	}

	return domains
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

type QoSController struct {
	targetQPS int
	jitter    float64
	delayMin  time.Duration
	delayMax  time.Duration
}

func NewQoSController(cfg models.QoSConfig) *QoSController {
	if cfg.TargetQPS <= 0 {
		cfg.TargetQPS = 100
	}

	return &QoSController{
		targetQPS: cfg.TargetQPS,
		jitter:    cfg.Jitter,
		delayMin:  time.Duration(cfg.DelayMinMs) * time.Millisecond,
		delayMax:  time.Duration(cfg.DelayMaxMs) * time.Millisecond,
	}
}

func (q *QoSController) Wait() {
	baseInterval := time.Second / time.Duration(q.targetQPS)

	// jitter: +/- jitter% variation around the base interval
	// e.g. QPS=100 (10ms), jitter=0.1 => interval varies 9~11ms
	if q.jitter > 0 {
		jitterRange := time.Duration(float64(baseInterval) * q.jitter)
		if jitterRange > 0 {
			offset := time.Duration(rand.Int63n(int64(jitterRange*2+1))) - jitterRange
			baseInterval += offset
		}
	}
	if baseInterval < 0 {
		baseInterval = 0
	}
	time.Sleep(baseInterval)

	// extra per-packet delay (e.g. simulating network latency)
	if q.delayMax > 0 {
		delay := q.delayMin
		if q.delayMax > q.delayMin {
			delay += time.Duration(rand.Int63n(int64(q.delayMax - q.delayMin)))
		}
		time.Sleep(delay)
	}
}

func BuildEthernetFrame(dstMAC net.HardwareAddr, srcMAC net.HardwareAddr, payload []byte) []byte {
	frame := make([]byte, 14)
	copy(frame[0:6], dstMAC)
	copy(frame[6:12], srcMAC)
	frame[12] = 0x08
	frame[13] = 0x00
	return append(frame, payload...)
}

func BuildIPv4Packet(srcIP, dstIP string, payload []byte) ([]byte, error) {
	src := net.ParseIP(srcIP)
	dst := net.ParseIP(dstIP)
	if src == nil || dst == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	header := make([]byte, 20)
	header[0] = 0x45
	header[1] = 0x00
	totalLen := 20 + len(payload)
	binary.BigEndian.PutUint16(header[2:4], uint16(totalLen))
	binary.BigEndian.PutUint16(header[4:6], 0x4000)
	binary.BigEndian.PutUint16(header[6:8], 64)
	header[8] = 17
	binary.BigEndian.PutUint16(header[10:12], 0)

	copy(header[12:16], src.To4())
	copy(header[16:20], dst.To4())

	sum := checksum(header)
	binary.BigEndian.PutUint16(header[10:12], sum)

	return append(header, payload...), nil
}

func checksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}

func BuildUDPPacket(srcPort, dstPort uint16, payload []byte) []byte {
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], srcPort)
	binary.BigEndian.PutUint16(header[2:4], dstPort)
	binary.BigEndian.PutUint16(header[4:6], uint16(8+len(payload)))
	binary.BigEndian.PutUint16(header[6:8], 0)

	sumData := append(header, payload...)
	sum := udpChecksum(srcIP, dstIP, sumData)
	binary.BigEndian.PutUint16(header[6:8], sum)

	return append(header, payload...)
}

var srcIP, dstIP string

func udpChecksum(src, dst string, data []byte) uint16 {
	// Simplified - in production use proper checksum
	return 0
}

func ParseMAC(macStr string) (net.HardwareAddr, error) {
	return net.ParseMAC(macStr)
}