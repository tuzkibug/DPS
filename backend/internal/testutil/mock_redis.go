package testutil

import (
	"context"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

// MockRedis is a shared in-memory Redis mock for tests.
type MockRedis struct {
	Status    map[string]models.TaskStatus
	StartTime map[string]time.Time
	Stats     map[string]*models.TaskStats
}

func NewMockRedis() *MockRedis {
	return &MockRedis{
		Status:    make(map[string]models.TaskStatus),
		StartTime: make(map[string]time.Time),
		Stats:     make(map[string]*models.TaskStats),
	}
}

func (m *MockRedis) SetTaskStatus(_ context.Context, id uuid.UUID, s models.TaskStatus) error {
	m.Status[id.String()] = s
	return nil
}

func (m *MockRedis) GetTaskStatus(_ context.Context, id uuid.UUID) (models.TaskStatus, error) {
	return m.Status[id.String()], nil
}

func (m *MockRedis) SetTaskStats(_ context.Context, stats *models.TaskStats) error {
	m.Stats[stats.TaskID.String()] = stats
	return nil
}

func (m *MockRedis) GetTaskStats(_ context.Context, id uuid.UUID) (*models.TaskStats, error) {
	return m.Stats[id.String()], nil
}

func (m *MockRedis) SetStartTime(_ context.Context, id uuid.UUID, t time.Time) error {
	m.StartTime[id.String()] = t
	return nil
}

func (m *MockRedis) GetStartTime(_ context.Context, id uuid.UUID) (time.Time, error) {
	t, ok := m.StartTime[id.String()]
	if !ok {
		return time.Time{}, nil
	}
	return t, nil
}

func (m *MockRedis) ClearTaskData(_ context.Context, id uuid.UUID) error {
	delete(m.Status, id.String())
	delete(m.StartTime, id.String())
	delete(m.Stats, id.String())
	return nil
}
