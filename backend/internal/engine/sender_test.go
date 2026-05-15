package engine

import (
	"context"
	"testing"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

func makeTestTask(qps int) *models.Task {
	return &models.Task{
		ID:      uuid.New(),
		Name:    "test",
		SrcIP:   "192.168.1.1",
		DstIP:   "8.8.8.8",
		SrcMAC:  "aa:bb:cc:dd:ee:ff",
		DstMAC:  "11:22:33:44:55:66",
		QoS:     models.QoSConfig{TargetQPS: qps},
	}
}

func TestNewPacketSender_InvalidSrcMAC(t *testing.T) {
	task := makeTestTask(100)
	task.SrcMAC = "bad"
	_, err := NewPacketSender(task, []string{"example.com"})
	if err == nil {
		t.Error("expected error for invalid source MAC, got nil")
	}
}

func TestNewPacketSender_InvalidDstMAC(t *testing.T) {
	task := makeTestTask(100)
	task.DstMAC = "bad"
	_, err := NewPacketSender(task, []string{"example.com"})
	if err == nil {
		t.Error("expected error for invalid destination MAC, got nil")
	}
}

func TestNewPacketSender_Success(t *testing.T) {
	task := makeTestTask(100)
	s, err := NewPacketSender(task, []string{"example.com", "google.com"})
	if err != nil {
		t.Fatalf("NewPacketSender failed: %v", err)
	}
	defer s.Stop()

	if len(s.domains) != 2 {
		t.Errorf("domains = %d, want 2", len(s.domains))
	}
	if s.qos.BatchSize() != 1 {
		t.Errorf("BatchSize = %d, want 1", s.qos.BatchSize())
	}
}

func TestNewPacketSender_HighQPSBatch(t *testing.T) {
	task := makeTestTask(10000)
	s, err := NewPacketSender(task, []string{"example.com"})
	if err != nil {
		t.Fatalf("NewPacketSender failed: %v", err)
	}
	defer s.Stop()

	if s.qos.BatchSize() <= 1 {
		t.Errorf("BatchSize = %d, expected > 1 for QPS=10000", s.qos.BatchSize())
	}
}

func TestPacketSender_Start_Stop(t *testing.T) {
	task := makeTestTask(100)
	s, err := NewPacketSender(task, []string{"example.com"})
	if err != nil {
		t.Fatalf("NewPacketSender failed: %v", err)
	}

	statsCh := make(chan *models.TaskStats, 10)
	ctx := context.Background()

	s.Start(ctx, task.ID, statsCh)
	// Give it a moment to start sending
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	// Stats should have been emitted
	select {
	case stats := <-statsCh:
		if stats.Status != models.TaskStatusRunning {
			t.Errorf("stats.Status = %s, want running", stats.Status)
		}
		if stats.TaskID != task.ID {
			t.Errorf("stats.TaskID mismatch")
		}
	default:
		// No stats received is also fine if we stopped quickly
	}
}

func TestPacketSender_Run_ContextCancel(t *testing.T) {
	task := makeTestTask(100)
	s, err := NewPacketSender(task, []string{"example.com"})
	if err != nil {
		t.Fatalf("NewPacketSender failed: %v", err)
	}

	statsCh := make(chan *models.TaskStats, 10)
	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx, task.ID, statsCh)

	// Cancel after a brief delay
	time.Sleep(30 * time.Millisecond)
	cancel()

	// Should stop without deadlock (s.wg.Done should fire)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
		// Clean up the connection ourselves since Stop also calls conn.Close
		s.conn.Close()
	}()
	select {
	case <-done:
		// clean shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("sender did not stop after context cancel within timeout")
	}
}

func TestPacketSender_sendPacket_NoDomains(t *testing.T) {
	s := &PacketSender{
		domains: []string{},
		dstIP:   []byte{8, 8, 8, 8},
	}
	err := s.sendPacket()
	if err == nil {
		t.Error("expected error with empty domains, got nil")
	}
}

func TestPacketSender_sendPacket_WithDomains(t *testing.T) {
	task := makeTestTask(100)
	s, err := NewPacketSender(task, []string{"example.com", "google.com", "cloudflare.com"})
	if err != nil {
		t.Fatalf("NewPacketSender failed: %v", err)
	}
	defer s.Stop()

	// All three sends should succeed
	for i := 0; i < 10; i++ {
		if err := s.sendPacket(); err != nil {
			t.Fatalf("sendPacket iteration %d failed: %v", i, err)
		}
	}
}

func TestPCAPSender_Run_EmptyGroups(t *testing.T) {
	sender := &PCAPSender{
		groups:    nil,
		totalPkts: 0,
		stopCh:    make(chan struct{}),
		qos:       NewQoSController(models.QoSConfig{TargetQPS: 100}),
	}

	statsCh := make(chan *models.TaskStats, 10)
	ctx := context.Background()
	id := uuid.New()

	sender.wg.Add(1)
	sender.run(ctx, id, statsCh)
	// Should return immediately due to empty groups check
}

func TestNewPCAPSender_InvalidIP(t *testing.T) {
	task := makeTestTask(100)
	task.SrcIP = "invalid"
	_, err := NewPCAPSender(task, "/nonexistent")
	if err == nil {
		t.Error("expected error for invalid source IP, got nil")
	}
}

func TestNewPCAPSender_InvalidPath(t *testing.T) {
	task := makeTestTask(100)
	_, err := NewPCAPSender(task, "/nonexistent/pcap/dir")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

func TestNewPCAPSender_InvalidDstMAC(t *testing.T) {
	task := makeTestTask(100)
	task.DstMAC = "badmac"
	_, err := NewPCAPSender(task, "/nonexistent")
	if err == nil {
		t.Error("expected error for invalid dst MAC, got nil")
	}
}

func TestNewPCAPSender_InvalidDstIP(t *testing.T) {
	task := makeTestTask(100)
	task.DstIP = "invalid"
	_, err := NewPCAPSender(task, "/nonexistent")
	if err == nil {
		t.Error("expected error for invalid dst IP, got nil")
	}
}

func TestNewPCAPSender_IPv6DstIP(t *testing.T) {
	task := makeTestTask(100)
	task.DstIP = "2001:db8::1"
	_, err := NewPCAPSender(task, "/nonexistent")
	if err == nil {
		t.Error("expected error for IPv6 dst IP, got nil")
	}
}

func TestNewPacketSender_RawMode_InvalidInterface(t *testing.T) {
	task := makeTestTask(100)
	task.RandomSrcIP = true
	task.Interface = "nonexistent12345"
	_, err := NewPacketSender(task, []string{"example.com"})
	if err == nil {
		t.Error("expected error for nonexistent interface in raw mode, got nil")
	}
}

func TestNewPacketSender_RawMode_InvalidSrcIP(t *testing.T) {
	task := makeTestTask(100)
	task.RandomSrcIP = true
	task.SrcIP = "invalid"
	_, err := NewPacketSender(task, []string{"example.com"})
	if err == nil {
		t.Error("expected error for invalid src IP in raw mode, got nil")
	}
}

func TestNewPacketSender_RawMode_InvalidDstIP(t *testing.T) {
	task := makeTestTask(100)
	task.RandomSrcIP = true
	task.Interface = "lo"
	task.DstIP = "invalid"
	_, err := NewPacketSender(task, []string{"example.com"})
	if err == nil {
		t.Error("expected error for invalid dst IP in raw mode, got nil")
	}
}
