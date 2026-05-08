package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

type PacketSender struct {
	conn       net.PacketConn
	srcMAC     net.HardwareAddr
	dstMAC     net.HardwareAddr
	srcIP      net.IP
	dstIP      net.IP
	qos        *QoSController
	domains    []string
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewPacketSender(cfg *models.Task, domains []string) (*PacketSender, error) {
	srcMAC, err := net.ParseMAC(cfg.SrcMAC)
	if err != nil {
		return nil, fmt.Errorf("invalid source MAC: %w", err)
	}

	dstMAC, err := net.ParseMAC(cfg.DstMAC)
	if err != nil {
		return nil, fmt.Errorf("invalid destination MAC: %w", err)
	}

	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("failed to open raw socket: %w", err)
	}

	return &PacketSender{
		conn:     conn,
		srcMAC:   srcMAC,
		dstMAC:   dstMAC,
		srcIP:    net.ParseIP(cfg.SrcIP),
		dstIP:    net.ParseIP(cfg.DstIP),
		qos:      NewQoSController(cfg.QoS),
		domains:  domains,
		stopCh:   make(chan struct{}),
	}, nil
}

func (s *PacketSender) Start(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	s.wg.Add(1)
	go s.run(ctx, taskID, statsChan)
}

func (s *PacketSender) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.conn.Close()
}

func (s *PacketSender) run(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var sentCount int64
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			stats := &models.TaskStats{
				TaskID:     taskID,
				SentCount:  sentCount,
				CurrentQPS: float64(sentCount),
				StartTime:  startTime,
				ElapsedMs:   time.Since(startTime).Milliseconds(),
				Status:      models.TaskStatusRunning,
			}
			select {
			case statsChan <- stats:
			default:
			}
			sentCount = 0
		default:
			s.sendPacket()
			sentCount++
			s.qos.Wait()
		}
	}
}

func (s *PacketSender) sendPacket() error {
	if len(s.domains) == 0 {
		return fmt.Errorf("no domains available")
	}
	domain := s.domains[time.Now().UnixNano()%int64(len(s.domains))]
	txID := uint16(time.Now().UnixNano() & 0xFFFF)

	dnsQuery := BuildDNSQuery(domain, txID)

	dstAddr := net.UDPAddr{IP: s.dstIP, Port: 53}
	_, err := s.conn.WriteTo(dnsQuery, &dstAddr)
	return err
}

type PCAPSender struct {
	conn     net.PacketConn
	pcapFile string
	cfg      *models.Task
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewPCAPSender(cfg *models.Task, pcapFile string) (*PCAPSender, error) {
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("failed to open raw socket: %w", err)
	}

	return &PCAPSender{
		conn:     conn,
		pcapFile: pcapFile,
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}, nil
}

func (p *PCAPSender) Start(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	p.wg.Add(1)
	go p.run(ctx, taskID, statsChan)
}

func (p *PCAPSender) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	p.conn.Close()
}

func (p *PCAPSender) run(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var sentCount int64
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			stats := &models.TaskStats{
				TaskID:     taskID,
				SentCount:  sentCount,
				CurrentQPS: float64(sentCount),
				StartTime:  startTime,
				ElapsedMs:  time.Since(startTime).Milliseconds(),
				Status:     models.TaskStatusRunning,
			}
			select {
			case statsChan <- stats:
			default:
			}
			sentCount = 0
		default:
			time.Sleep(time.Second / time.Duration(p.cfg.QoS.TargetQPS))
			sentCount++
		}
	}
}

func (p *PCAPSender) ReadPCAPFile(file string) ([][]byte, error) {
	return nil, fmt.Errorf("PCAP reading not implemented")
}

func (p *PCAPSender) sendPacket(packetData []byte) error {
	dstAddr := net.UDPAddr{IP: net.ParseIP(p.cfg.DstIP), Port: 53}
	_, err := p.conn.WriteTo(packetData, &dstAddr)
	return err
}