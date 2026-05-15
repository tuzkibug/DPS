package engine

import (
	"context"
	"fmt"
	"math/rand/v2"
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
	rawMode    bool
	randomSrcIP  bool
	randomSrcMAC bool
	rawSockfd   int
	srcPort     uint16
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

	dstIP := net.ParseIP(cfg.DstIP)
	if dstIP == nil {
		return nil, fmt.Errorf("invalid destination IP: %s", cfg.DstIP)
	}

	useRaw := cfg.RandomSrcIP || cfg.RandomSrcMAC || cfg.Interface != ""
	sender := &PacketSender{
		srcMAC:      srcMAC,
		dstMAC:      dstMAC,
		srcIP:       net.ParseIP(cfg.SrcIP),
		dstIP:       dstIP,
		rawMode:     useRaw,
		randomSrcIP: cfg.RandomSrcIP,
		randomSrcMAC: cfg.RandomSrcMAC,
		srcPort:     uint16(rand.Uint32()),
		qos:         NewQoSController(cfg.QoS),
		domains:     domains,
		stopCh:      make(chan struct{}),
		createdAt:   cfg.CreatedAt,
		lastRunAt:   cfg.LastRunAt,
		totalRunMs:  cfg.TotalRunMs,
	}

	if useRaw {
		var fd int
		var err error
		if cfg.Interface != "" {
			fd, _, err = openRawSocketByName(cfg.Interface)
		} else {
			srcIP := net.ParseIP(cfg.SrcIP)
			if srcIP == nil {
				return nil, fmt.Errorf("invalid source IP: %s", cfg.SrcIP)
			}
			fd, _, err = openRawSocket(srcIP)
		}
		if err != nil {
			return nil, fmt.Errorf("open raw socket: %w", err)
		}
		sender.rawSockfd = fd
	} else {
		conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			return nil, fmt.Errorf("failed to open socket: %w", err)
		}
		sender.conn = conn
	}

	return sender, nil
}

func (s *PacketSender) Start(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats) {
	s.wg.Add(1)
	go s.run(ctx, taskID, statsChan)
}

func (s *PacketSender) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	if s.rawMode {
		CloseRaw(s.rawSockfd)
	} else {
		s.conn.Close()
	}
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
				if err := s.sendPacket(); err != nil {
					continue
				}
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
	domain := s.domains[rand.IntN(len(s.domains))]
	txID := uint16(rand.Uint32())

	dnsQuery, err := BuildDNSQuery(domain, txID)
	if err != nil {
		return err
	}

	if s.rawMode {
		srcIP := s.srcIP
		if s.randomSrcIP {
			srcIP = randomIPv4()
		}
		srcMAC := s.srcMAC
		if s.randomSrcMAC {
			srcMAC = randomMAC()
		}
		srcIPStr := srcIP.String()
		dstIPStr := s.dstIP.String()

		udpPayload := BuildUDPPacket(s.srcPort, 53, srcIPStr, dstIPStr, dnsQuery)
		ipPacket, err := BuildIPv4Packet(srcIPStr, dstIPStr, udpPayload)
		if err != nil {
			return err
		}
		frame := BuildEthernetFrame(s.dstMAC, srcMAC, ipPacket)
		return WriteRaw(s.rawSockfd, frame)
	}

	dstAddr := net.UDPAddr{IP: s.dstIP, Port: 53}
	_, err = s.conn.WriteTo(dnsQuery, &dstAddr)
	return err
}

type PCAPSender struct {
	sockfd     int
	cfg        *models.Task
	groups     []packetGroup
	rawGroups  []packetGroup
	totalPkts  int
	curGroup   int
	curPkt     int
	randomSrcIP  bool
	randomSrcMAC bool
	dstMAC     net.HardwareAddr
	dstIP      net.IP
	qos        *QoSController
	stopCh     chan struct{}
	wg         sync.WaitGroup
	createdAt  time.Time
	lastRunAt  *time.Time
	totalRunMs int64
}

func NewPCAPSender(cfg *models.Task, pcapDir string) (*PCAPSender, error) {
	var fd int
	var err error
	if cfg.Interface != "" {
		fd, _, err = openRawSocketByName(cfg.Interface)
	} else {
		srcIP := net.ParseIP(cfg.SrcIP)
		if srcIP == nil {
			return nil, fmt.Errorf("invalid source IP: %s", cfg.SrcIP)
		}
		fd, _, err = openRawSocket(srcIP)
	}
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	randomize := cfg.RandomSrcIP || cfg.RandomSrcMAC
	var groups, rawGroups []packetGroup

	if randomize {
		rawGroups, err = loadRawPacketsFromPath(pcapDir)
		if err != nil {
			CloseRaw(fd)
			return nil, fmt.Errorf("load pcap: %w", err)
		}
	} else {
		groups, err = loadPacketsFromPath(pcapDir, cfg.SrcMAC, cfg.DstMAC, cfg.SrcIP, cfg.DstIP)
		if err != nil {
			CloseRaw(fd)
			return nil, fmt.Errorf("load pcap: %w", err)
		}
	}

	totalPkts := 0
	for _, g := range groups {
		totalPkts += len(g.Packets)
	}
	for _, g := range rawGroups {
		totalPkts += len(g.Packets)
	}

	dstMAC, _ := net.ParseMAC(cfg.DstMAC)
	dstIP := net.ParseIP(cfg.DstIP)

	return &PCAPSender{
		sockfd:     fd,
		cfg:        cfg,
		groups:     groups,
		rawGroups:  rawGroups,
		totalPkts:  totalPkts,
		randomSrcIP:  cfg.RandomSrcIP,
		randomSrcMAC: cfg.RandomSrcMAC,
		dstMAC:     dstMAC,
		dstIP:      dstIP,
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

	useRaw := len(p.rawGroups) > 0
	usePreRewritten := len(p.groups) > 0
	if !useRaw && !usePreRewritten {
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
				var pkt []byte
				if useRaw {
					grp := p.rawGroups[p.curGroup]
					if p.curPkt >= len(grp.Packets) {
						p.advance()
						continue
					}
					raw := grp.Packets[p.curPkt]
					srcIP := net.ParseIP(p.cfg.SrcIP)
					if p.randomSrcIP {
						srcIP = randomIPv4()
					}
					srcMAC, _ := net.ParseMAC(p.cfg.SrcMAC)
					if p.randomSrcMAC {
						srcMAC = randomMAC()
					}
					var err error
					pkt, err = rewritePacket(raw, srcMAC, p.dstMAC, srcIP, p.dstIP)
					if err != nil {
						p.advance()
						continue
					}
				} else {
					grp := p.groups[p.curGroup]
					if p.curPkt >= len(grp.Packets) {
						p.advance()
						continue
					}
					pkt = grp.Packets[p.curPkt]
				}

				if err := WriteRaw(p.sockfd, pkt); err != nil {
					return
				}
				sentCount++
				p.advance()
			}
			p.qos.Wait()
		}
	}
}

func (p *PCAPSender) advance() {
	p.curPkt++
	grps := p.groups
	if len(p.rawGroups) > 0 {
		grps = p.rawGroups
	}
	if p.curPkt >= len(grps[p.curGroup].Packets) {
		p.curPkt = 0
		p.curGroup++
		if p.curGroup >= len(grps) {
			p.curGroup = 0
		}
	}
}

func (p *PCAPSender) sendPacket(packetData []byte) error {
	return WriteRaw(p.sockfd, packetData)
}