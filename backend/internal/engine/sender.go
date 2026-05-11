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
	createdAt  time.Time
	lastRunAt  *time.Time
	totalRunMs int64
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
		conn:       conn,
		srcMAC:     srcMAC,
		dstMAC:     dstMAC,
		srcIP:      net.ParseIP(cfg.SrcIP),
		dstIP:      net.ParseIP(cfg.DstIP),
		qos:        NewQoSController(cfg.QoS),
		domains:    domains,
		stopCh:     make(chan struct{}),
		createdAt:  cfg.CreatedAt,
		lastRunAt:  cfg.LastRunAt,
		totalRunMs: cfg.TotalRunMs,
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
				CreatedAt:   s.createdAt,
				LastRunAt:   s.lastRunAt,
				TotalRunMs:  s.totalRunMs,
			}
			select {
			case statsChan <- stats:
			default:
			}
			sentCount = 0
		default:
			batch := s.qos.BatchSize()
			for i := 0; i < batch; i++ {
				s.sendPacket()
				sentCount++
			}
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
	sockfd     int
	cfg        *models.Task
	groups     []packetGroup
	totalPkts  int
	curGroup   int
	curPkt     int
	qos        *QoSController
	stopCh     chan struct{}
	wg         sync.WaitGroup
	createdAt  time.Time
	lastRunAt  *time.Time
	totalRunMs int64
}

func NewPCAPSender(cfg *models.Task, pcapDir string) (*PCAPSender, error) {
	srcIP := net.ParseIP(cfg.SrcIP)
	if srcIP == nil {
		return nil, fmt.Errorf("invalid source IP: %s", cfg.SrcIP)
	}

	fd, _, err := openRawSocket(srcIP)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	groups, err := loadPacketsFromPath(pcapDir, cfg.SrcMAC, cfg.DstMAC, cfg.SrcIP, cfg.DstIP)
	if err != nil {
		CloseRaw(fd)
		return nil, fmt.Errorf("load pcap: %w", err)
	}

	totalPkts := 0
	for _, g := range groups {
		totalPkts += len(g.Packets)
	}

	return &PCAPSender{
		sockfd:     fd,
		cfg:        cfg,
		groups:     groups,
		totalPkts:  totalPkts,
		qos:        NewQoSController(cfg.QoS),
		stopCh:     make(chan struct{}),
		createdAt:  cfg.CreatedAt,
		lastRunAt:  cfg.LastRunAt,
		totalRunMs: cfg.TotalRunMs,
	}, nil
}

func (p *PCAPSender) Start(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	p.wg.Add(1)
	go p.run(ctx, taskID, statsChan)
}

func (p *PCAPSender) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	CloseRaw(p.sockfd)
}

func (p *PCAPSender) run(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	defer p.wg.Done()

	if len(p.groups) == 0 || p.totalPkts == 0 {
		return
	}

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
				CreatedAt:  p.createdAt,
				LastRunAt:  p.lastRunAt,
				TotalRunMs: p.totalRunMs,
			}
			select {
			case statsChan <- stats:
			default:
			}
			sentCount = 0
		default:
			batch := p.qos.BatchSize()
			for i := 0; i < batch; i++ {
				grp := p.groups[p.curGroup]
				if len(grp.Packets) > 0 {
					if err := WriteRaw(p.sockfd, grp.Packets[p.curPkt]); err != nil {
						return
					}
					sentCount++
				}

				p.curPkt++
				if p.curPkt >= len(grp.Packets) {
					p.curPkt = 0
					p.curGroup++
					if p.curGroup >= len(p.groups) {
						p.curGroup = 0
					}
				}
			}
			p.qos.Wait()
		}
	}
}

func (p *PCAPSender) ReadPCAPFile(file string) ([][]byte, error) {
	return nil, fmt.Errorf("PCAP reading not implemented")
}

func (p *PCAPSender) sendPacket(packetData []byte) error {
	return WriteRaw(p.sockfd, packetData)
}