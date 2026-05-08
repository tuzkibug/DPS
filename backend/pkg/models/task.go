package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusStopped   TaskStatus = "stopped"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type InputType string

const (
	InputTypeCSV   InputType = "csv"
	InputTypePCAP  InputType = "pcap"
)

type QoSConfig struct {
	TargetQPS    int     `json:"target_qps"`    // 目标QPS
	Jitter       float64 `json:"jitter"`        // 抖动比例 0.0-1.0
	DelayMinMs   int     `json:"delay_min_ms"`  // 最小延迟(毫秒)
	DelayMaxMs   int     `json:"delay_max_ms"`  // 最大延迟(毫秒)
	burstSize    int     `json:"burst_size"`    //  burst size
}

type Task struct {
	ID           uuid.UUID   `json:"id"`
	Name         string      `json:"name"`
	InputType    InputType   `json:"input_type"`
	FilePath     string      `json:"file_path"`      // CSV or PCAP file path
	SrcIP        string      `json:"src_ip"`
	DstIP        string      `json:"dst_ip"`
	SrcMAC       string      `json:"src_mac"`
	DstMAC       string      `json:"dst_mac"`
	StartTime    time.Time   `json:"start_time"`     // 计划开始时间
	DurationMs   int         `json:"duration_ms"`    // 持续时间(毫秒), 0表示无限
	QoS          QoSConfig   `json:"qos"`
	Status       TaskStatus  `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type TaskStats struct {
	TaskID       uuid.UUID `json:"task_id"`
	SentCount    int64     `json:"sent_count"`
	FailedCount  int64     `json:"failed_count"`
	CurrentQPS   float64   `json:"current_qps"`
	StartTime    time.Time `json:"start_time"`
	ElapsedMs    int64     `json:"elapsed_ms"`
	Status       TaskStatus `json:"status"`
}

type CreateTaskRequest struct {
	Name        string     `json:"name" binding:"required"`
	InputType   InputType  `json:"input_type" binding:"required"`
	FileContent string     `json:"file_content"` // base64 encoded file content
	SrcIP       string     `json:"src_ip" binding:"required"`
	DstIP       string     `json:"dst_ip" binding:"required"`
	SrcMAC      string     `json:"src_mac" binding:"required"`
	DstMAC      string     `json:"dst_mac" binding:"required"`
	StartTime   *time.Time `json:"start_time"`
	DurationMs  int        `json:"duration_ms"`
	QoS         QoSConfig  `json:"qos" binding:"required"`
}

type UpdateTaskRequest struct {
	Name        *string    `json:"name"`
	SrcIP       *string    `json:"src_ip"`
	DstIP       *string    `json:"dst_ip"`
	SrcMAC      *string    `json:"src_mac"`
	DstMAC      *string    `json:"dst_mac"`
	StartTime   *time.Time `json:"start_time"`
	DurationMs  *int       `json:"duration_ms"`
	QoS         *QoSConfig `json:"qos"`
}

type WSMessage struct {
	Type string      `json:"type"` // "stats", "status_change", "error"
	Data interface{} `json:"data"`
}