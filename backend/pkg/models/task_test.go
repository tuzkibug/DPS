package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTaskStatusConstants(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusStopped,
		TaskStatusCompleted,
		TaskStatusFailed,
	}

	expected := []string{"pending", "running", "stopped", "completed", "failed"}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("TaskStatus = %q, want %q", status, expected[i])
		}
	}
}

func TestInputTypeConstants(t *testing.T) {
	types := []InputType{InputTypeCSV, InputTypePCAP}
	expected := []string{"csv", "pcap"}

	for i, typ := range types {
		if string(typ) != expected[i] {
			t.Errorf("InputType = %q, want %q", typ, expected[i])
		}
	}
}

func TestQoSConfigJSON(t *testing.T) {
	qos := QoSConfig{
		TargetQPS:  500,
		Jitter:     0.15,
		DelayMinMs: 5,
		DelayMaxMs: 50,
	}

	data, err := json.Marshal(qos)
	if err != nil {
		t.Fatalf("Failed to marshal QoSConfig: %v", err)
	}

	var unmarshaled QoSConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal QoSConfig: %v", err)
	}

	if unmarshaled.TargetQPS != qos.TargetQPS {
		t.Errorf("TargetQPS = %d, want %d", unmarshaled.TargetQPS, qos.TargetQPS)
	}
	if unmarshaled.Jitter != qos.Jitter {
		t.Errorf("Jitter = %f, want %f", unmarshaled.Jitter, qos.Jitter)
	}
}

func TestTaskJSON(t *testing.T) {
	taskID := uuid.New()
	now := time.Now()

	task := Task{
		ID:         taskID,
		Name:       "Test Task",
		InputType:  InputTypeCSV,
		FilePath:   "/path/to/file.csv",
		SrcIP:      "192.168.1.100",
		DstIP:      "8.8.8.8",
		SrcMAC:     "aa:bb:cc:dd:ee:ff",
		DstMAC:     "11:22:33:44:55:66",
		StartTime:  now,
		DurationMs: 60000,
		QoS: QoSConfig{
			TargetQPS:  100,
			Jitter:     0.1,
			DelayMinMs: 0,
			DelayMaxMs: 100,
		},
		Status:    TaskStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal Task: %v", err)
	}

	var unmarshaled Task
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Task: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("ID = %v, want %v", unmarshaled.ID, task.ID)
	}
	if unmarshaled.Name != task.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, task.Name)
	}
	if unmarshaled.Status != task.Status {
		t.Errorf("Status = %q, want %q", unmarshaled.Status, task.Status)
	}
}

func TestCreateTaskRequestValidation(t *testing.T) {
	jsonData := `{
		"name": "Test Task",
		"input_type": "csv",
		"src_ip": "192.168.1.100",
		"dst_ip": "8.8.8.8",
		"src_mac": "aa:bb:cc:dd:ee:ff",
		"dst_mac": "11:22:33:44:55:66",
		"qos": {
			"target_qps": 100,
			"jitter": 0.1,
			"delay_min_ms": 0,
			"delay_max_ms": 100
		}
	}`

	var req CreateTaskRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("Failed to unmarshal CreateTaskRequest: %v", err)
	}

	if req.Name != "Test Task" {
		t.Errorf("Name = %q, want %q", req.Name, "Test Task")
	}
	if req.InputType != InputTypeCSV {
		t.Errorf("InputType = %q, want %q", req.InputType, InputTypeCSV)
	}
	if req.QoS.TargetQPS != 100 {
		t.Errorf("QoS.TargetQPS = %d, want 100", req.QoS.TargetQPS)
	}
}

func TestWSMessageJSON(t *testing.T) {
	msg := WSMessage{
		Type: "stats",
		Data: map[string]interface{}{
			"sent_count": 1000,
			"current_qps": 95.5,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal WSMessage: %v", err)
	}

	var unmarshaled WSMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal WSMessage: %v", err)
	}

	if unmarshaled.Type != "stats" {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, "stats")
	}
}

func TestTaskStatsFields(t *testing.T) {
	stats := TaskStats{
		TaskID:     uuid.New(),
		SentCount:  5000,
		FailedCount: 10,
		CurrentQPS: 95.5,
		StartTime:  time.Now(),
		ElapsedMs:  30000,
		Status:     TaskStatusRunning,
	}

	if stats.SentCount != 5000 {
		t.Errorf("SentCount = %d, want 5000", stats.SentCount)
	}
	if stats.CurrentQPS != 95.5 {
		t.Errorf("CurrentQPS = %f, want 95.5", stats.CurrentQPS)
	}
}