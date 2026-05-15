package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"dns-sender/internal/engine"
	"dns-sender/internal/store"
	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

type RedisOps interface {
	SetTaskStatus(ctx context.Context, taskID uuid.UUID, status models.TaskStatus) error
	GetTaskStatus(ctx context.Context, taskID uuid.UUID) (models.TaskStatus, error)
	SetTaskStats(ctx context.Context, stats *models.TaskStats) error
	GetTaskStats(ctx context.Context, taskID uuid.UUID) (*models.TaskStats, error)
	SetStartTime(ctx context.Context, taskID uuid.UUID, t time.Time) error
	GetStartTime(ctx context.Context, taskID uuid.UUID) (time.Time, error)
	ClearTaskData(ctx context.Context, taskID uuid.UUID) error
}

type TaskScheduler struct {
	tasks    map[uuid.UUID]*TaskInfo
	mu       sync.RWMutex
	sqlite   *store.SQLiteStore
	redis    RedisOps
}

type TaskInfo struct {
	ID       uuid.UUID
	Sender   interface {
		Start(ctx context.Context, taskID uuid.UUID, statsChan chan<- *models.TaskStats)
		Stop()
	}
	Cancel   context.CancelFunc
	StatsChan chan *models.TaskStats
}

func NewTaskScheduler(sqlite *store.SQLiteStore, redis RedisOps) *TaskScheduler {
	s := &TaskScheduler{
		tasks:  make(map[uuid.UUID]*TaskInfo),
		sqlite: sqlite,
		redis:  redis,
	}
	s.recoverTasks()
	return s
}

func (s *TaskScheduler) recoverTasks() {
	tasks, err := s.sqlite.ListTasks()
	if err != nil {
		log.Printf("recoverTasks: failed to list tasks: %v", err)
		return
	}
	for _, task := range tasks {
		if task.Status == models.TaskStatusRunning {
			log.Printf("recoverTasks: recovering task %s (%s)", task.ID, task.Name)
			if err := s.StartTask(task); err != nil {
				log.Printf("recoverTasks: failed to recover task %s: %v", task.ID, err)
				task.Status = models.TaskStatusPending
				if uerr := s.sqlite.UpdateTask(task); uerr != nil {
					log.Printf("recoverTasks: failed to update task %s status: %v", task.ID, uerr)
				}
				s.redis.SetTaskStatus(context.Background(), task.ID, models.TaskStatusPending)
			}
		}
	}
}

func (s *TaskScheduler) StartTask(task *models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	statsChan := make(chan *models.TaskStats, 100)

	info := &TaskInfo{
		ID:        task.ID,
		Cancel:   cancel,
		StatsChan: statsChan,
	}

	switch task.InputType {
	case models.InputTypeCSV:
		var domains []string
		if task.FilePath != "" {
			domains = engine.ParseCSVDomains(s.readFileContent(task.FilePath))
		}
		if len(domains) == 0 {
			log.Printf("StartTask: no domains for task %s, using fallback defaults", task.ID)
			domains = []string{"example.com", "google.com", "cloudflare.com"}
		}
		sender, err := engine.NewPacketSender(task, domains)
		if err != nil {
			return err
		}
		info.Sender = sender
		sender.Start(ctx, task.ID, statsChan)

	case models.InputTypePCAP:
		sender, err := engine.NewPCAPSender(task, task.FilePath)
		if err != nil {
			return err
		}
		info.Sender = sender
		sender.Start(ctx, task.ID, statsChan)
	}

	s.tasks[task.ID] = info

	go s.watchStats(task.ID, statsChan)

	now := time.Now()
	task.Status = models.TaskStatusRunning
	task.LastRunAt = &now
	s.sqlite.UpdateTask(task)
	s.redis.SetTaskStatus(context.Background(), task.ID, models.TaskStatusRunning)
	s.redis.SetStartTime(context.Background(), task.ID, now)

	// Auto-stop after fixed duration if set
	if task.DurationMs > 0 {
		dur := time.Duration(task.DurationMs) * time.Millisecond
		time.AfterFunc(dur, func() {
			log.Printf("auto-stopping task %s after duration %v", task.ID, dur)
			s.StopTask(task.ID)
		})
	}

	return nil
}

func (s *TaskScheduler) StopTask(taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.tasks[taskID]
	if !exists {
		return nil
	}

	info.Cancel()
	if info.Sender != nil {
		info.Sender.Stop()
	}
	// Close stats channel so watchStats goroutine exits before we zero Redis
	close(info.StatsChan)
	delete(s.tasks, taskID)

	// Zero out QPS in Redis stats so clients see the drop immediately
	stats, err := s.redis.GetTaskStats(context.Background(), taskID)
	if err == nil {
		stats.CurrentQPS = 0
		stats.Status = models.TaskStatusPending
		s.redis.SetTaskStats(context.Background(), stats)
	}

	task, err := s.sqlite.GetTask(taskID)
	if err == nil {
		startTime, err := s.redis.GetStartTime(context.Background(), taskID)
		if err != nil {
			log.Printf("StopTask: failed to get start time for task %s: %v", taskID, err)
		}
		if !startTime.IsZero() {
			elapsed := time.Since(startTime).Milliseconds()
			task.TotalRunMs += elapsed
		}
		task.Status = models.TaskStatusPending
		s.sqlite.UpdateTask(task)
	} else {
		log.Printf("StopTask: failed to get task %s from sqlite: %v", taskID, err)
	}
	s.redis.SetTaskStatus(context.Background(), taskID, models.TaskStatusPending)

	return nil
}

func (s *TaskScheduler) GetTaskStatus(taskID uuid.UUID) (models.TaskStatus, error) {
	return s.redis.GetTaskStatus(context.Background(), taskID)
}

func (s *TaskScheduler) GetStats(taskID uuid.UUID) (*models.TaskStats, error) {
	return s.redis.GetTaskStats(context.Background(), taskID)
}

func (s *TaskScheduler) watchStats(taskID uuid.UUID, statsChan chan *models.TaskStats) {
	for stats := range statsChan {
		if err := s.redis.SetTaskStats(context.Background(), stats); err != nil {
			log.Printf("watchStats: failed to set stats for task %s: %v", taskID, err)
		}
	}
}

func (s *TaskScheduler) readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (s *TaskScheduler) ListTasks() []*models.Task {
	tasks, err := s.sqlite.ListTasks()
	if err != nil {
		return []*models.Task{}
	}
	if tasks == nil {
		return []*models.Task{}
	}
	return tasks
}

func (s *TaskScheduler) GetTask(taskID uuid.UUID) (*models.Task, error) {
	return s.sqlite.GetTask(taskID)
}

func (s *TaskScheduler) CreateTask(req *models.CreateTaskRequest, filePath string) (*models.Task, error) {
	task := &models.Task{
		ID:         uuid.New(),
		Name:       req.Name,
		InputType:  req.InputType,
		FilePath:   filePath,
		SrcIP:      req.SrcIP,
		DstIP:      req.DstIP,
		SrcMAC:     req.SrcMAC,
		DstMAC:     req.DstMAC,
		Interface:   req.Interface,
		RandomSrcIP:  req.RandomSrcIP,
		RandomSrcMAC: req.RandomSrcMAC,
		QoS:        req.QoS,
		Status:     models.TaskStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if req.StartTime != nil {
		task.StartTime = *req.StartTime
	}
	if req.DurationMs > 0 {
		task.DurationMs = req.DurationMs
	}

	err := s.sqlite.CreateTask(task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskScheduler) UpdateTask(taskID uuid.UUID, req *models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.sqlite.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.SrcIP != nil {
		task.SrcIP = *req.SrcIP
	}
	if req.DstIP != nil {
		task.DstIP = *req.DstIP
	}
	if req.SrcMAC != nil {
		task.SrcMAC = *req.SrcMAC
	}
	if req.DstMAC != nil {
		task.DstMAC = *req.DstMAC
	}
	if req.StartTime != nil {
		task.StartTime = *req.StartTime
	}
	if req.FilePath != nil {
		task.FilePath = *req.FilePath
	}
	if req.DurationMs != nil {
		task.DurationMs = *req.DurationMs
	}
	if req.QoS != nil {
		task.QoS = *req.QoS
	}

	task.UpdatedAt = time.Now()

	if task.RandomSrcIP && task.Interface == "" {
		return nil, fmt.Errorf("interface is required when random_src_ip is enabled")
	}

	err = s.sqlite.UpdateTask(task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskScheduler) DeleteTask(taskID uuid.UUID) error {
	s.StopTask(taskID)
	s.redis.ClearTaskData(context.Background(), taskID)
	return s.sqlite.DeleteTask(taskID)
}