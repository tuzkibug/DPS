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

func BuildDNSQuery(domain string, txID uint16) ([]byte, error) {
	header := make([]byte, 12)

	binary.BigEndian.PutUint16(header[0:2], txID)
	binary.BigEndian.PutUint16(header[2:4], 0x0100)
	binary.BigEndian.PutUint16(header[4:6], 1)
	binary.BigEndian.PutUint16(header[6:8], 0)
	binary.BigEndian.PutUint16(header[8:10], 0)
	binary.BigEndian.PutUint16(header[10:12], 0)

	labels, err := parseDomain(domain)
	if err != nil {
		return nil, err
	}
	query := append(header, labels...)

	tail := make([]byte, 4)
	binary.BigEndian.PutUint16(tail[0:2], 1) // QTYPE A
	binary.BigEndian.PutUint16(tail[2:4], 1) // QCLASS IN
	query = append(query, tail...)

	return query, nil
}

func parseDomain(domain string) ([]byte, error) {
	if len(domain) > 253 {
		return nil, fmt.Errorf("domain name exceeds 253 characters")
	}
	var result []byte
	labels := []byte{}
	for _, c := range []byte(domain) {
		if c == '.' {
			if len(labels) > 63 {
				return nil, fmt.Errorf("label exceeds 63 characters")
			}
			result = append(result, byte(len(labels)))
			result = append(result, labels...)
			labels = []byte{}
		} else {
			labels = append(labels, c)
		}
	}
	if len(labels) > 0 {
		if len(labels) > 63 {
			return nil, fmt.Errorf("label exceeds 63 characters")
		}
		result = append(result, byte(len(labels)))
		result = append(result, labels...)
	}
	result = append(result, 0)
	return result, nil
}

func ParseCSVDomains(content string) []string {
	var domains []string
	lines := []byte{}
	inQuote := false
	data := []byte(content)

	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == '"' {
			if inQuote && i+1 < len(data) && data[i+1] == '"' {
				lines = append(lines, '"')
				i++
			} else {
				inQuote = !inQuote
			}
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
	batchSize int
}

func NewQoSController(cfg models.QoSConfig) *QoSController {
	if cfg.TargetQPS <= 0 {
		cfg.TargetQPS = 100
	}

	interval := time.Second / time.Duration(cfg.TargetQPS)
	batchSize := 1
	if interval < time.Millisecond {
		batchSize = int(time.Millisecond / interval)
		if batchSize < 1 {
			batchSize = 1
		}
	}

	return &QoSController{
		targetQPS: cfg.TargetQPS,
		jitter:    cfg.Jitter,
		delayMin:  time.Duration(cfg.DelayMinMs) * time.Millisecond,
		delayMax:  time.Duration(cfg.DelayMaxMs) * time.Millisecond,
		batchSize: batchSize,
	}
}

func (q *QoSController) BatchSize() int {
	return q.batchSize
}

func (q *QoSController) Wait() {
	batchInterval := time.Second / time.Duration(q.targetQPS) * time.Duration(q.batchSize)

	if q.jitter > 0 {
		jitterRange := time.Duration(float64(batchInterval) * q.jitter)
		if jitterRange > 0 {
			offset := time.Duration(rand.Int63n(int64(jitterRange*2+1))) - jitterRange
			batchInterval += offset
		}
	}
	if batchInterval < 0 {
		batchInterval = 0
	}
	time.Sleep(batchInterval)

	if q.delayMax > 0 {
		for i := 0; i < q.batchSize; i++ {
			delay := q.delayMin
			if q.delayMax > q.delayMin {
				delay += time.Duration(rand.Int63n(int64(q.delayMax - q.delayMin)))
			}
			time.Sleep(delay)
		}
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

func BuildUDPPacket(srcPort, dstPort uint16, srcIP, dstIP string, payload []byte) []byte {
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

func udpChecksum(src, dst string, data []byte) uint16 {
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)
	if srcIP == nil || dstIP == nil {
		return 0
	}
	src4 := srcIP.To4()
	dst4 := dstIP.To4()
	if src4 == nil || dst4 == nil {
		return 0
	}

	pseudo := make([]byte, 12)
	copy(pseudo[0:4], src4)
	copy(pseudo[4:8], dst4)
	pseudo[9] = 17 // protocol UDP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(len(data)))

	return checksum(append(pseudo, data...))
}

func ParseMAC(macStr string) (net.HardwareAddr, error) {
	return net.ParseMAC(macStr)
}