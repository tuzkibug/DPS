package scheduler

import (
	"os"
	"testing"
	"time"

	"dns-sender/internal/store"
	"dns-sender/internal/testutil"
	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

func setupTestScheduler(t *testing.T) (*TaskScheduler, func()) {
	tmpfile, err := os.CreateTemp("", "test-sched.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	sqlite, err := store.NewSQLiteStore(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatal(err)
	}

	sched := NewTaskScheduler(sqlite, testutil.NewMockRedis())

	cleanup := func() {
		sqlite.Close()
		os.Remove(tmpfile.Name())
	}
	return sched, cleanup
}

func makeCreateReq() *models.CreateTaskRequest {
	return &models.CreateTaskRequest{
		Name:     "test-task",
		InputType: models.InputTypeCSV,
		SrcIP:    "192.168.1.1",
		DstIP:    "8.8.8.8",
		SrcMAC:   "aa:bb:cc:dd:ee:ff",
		DstMAC:   "11:22:33:44:55:66",
		QoS:      models.QoSConfig{TargetQPS: 100},
	}
}

func TestScheduler_CreateTask(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	task, err := sched.CreateTask(req, "/tmp/test.csv")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if task.Name != req.Name {
		t.Errorf("task.Name = %q, want %q", task.Name, req.Name)
	}
	if task.Status != models.TaskStatusPending {
		t.Errorf("task.Status = %s, want pending", task.Status)
	}
}

func TestScheduler_GetTask(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	created, _ := sched.CreateTask(req, "/tmp/test.csv")

	retrieved, err := sched.GetTask(created.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: %s vs %s", retrieved.ID, created.ID)
	}
}

func TestScheduler_GetTask_NotFound(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	_, err := sched.GetTask(uuid.New())
	if err == nil {
		t.Error("expected error for nonexistent task, got nil")
	}
}

func TestScheduler_ListTasks(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		req := makeCreateReq()
		req.Name = "task-" + string(rune('0'+i))
		sched.CreateTask(req, "/tmp/test.csv")
	}

	tasks := sched.ListTasks()
	if len(tasks) != 3 {
		t.Errorf("ListTasks returned %d tasks, want 3", len(tasks))
	}
}

func TestScheduler_ListTasks_Empty(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	tasks := sched.ListTasks()
	if len(tasks) != 0 {
		t.Errorf("ListTasks returned %d tasks, want 0", len(tasks))
	}
}

func TestScheduler_UpdateTask(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	task, _ := sched.CreateTask(req, "/tmp/test.csv")

	newName := "updated-name"
	newQPS := 500
	updateReq := &models.UpdateTaskRequest{
		Name: &newName,
		QoS:  &models.QoSConfig{TargetQPS: newQPS},
	}

	updated, err := sched.UpdateTask(task.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Name = %q, want %q", updated.Name, newName)
	}
	if updated.QoS.TargetQPS != newQPS {
		t.Errorf("TargetQPS = %d, want %d", updated.QoS.TargetQPS, newQPS)
	}
}

func TestScheduler_DeleteTask(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	task, _ := sched.CreateTask(req, "/tmp/test.csv")

	err := sched.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	_, err = sched.GetTask(task.ID)
	if err == nil {
		t.Error("expected error for deleted task, got nil")
	}
}

func TestScheduler_TrimQuotes(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	task, err := sched.CreateTask(req, `/tmp/"test".csv`)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if task.FilePath != `/tmp/"test".csv` {
		t.Errorf("FilePath = %q", task.FilePath)
	}
}

func TestScheduler_StartTask_InvalidCSV(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	task := &models.Task{
		ID:        uuid.New(),
		Name:      "broken",
		InputType: models.InputTypeCSV,
		FilePath:  "/nonexistent/file.csv",
		SrcIP:     "192.168.1.1",
		DstIP:     "8.8.8.8",
		SrcMAC:    "aa:bb:cc:dd:ee:ff",
		DstMAC:    "11:22:33:44:55:66",
		QoS:       models.QoSConfig{TargetQPS: 100},
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// StartTask will use default domains when file is missing
	err := sched.StartTask(task)
	if err != nil {
		t.Fatalf("StartTask should succeed with default domains: %v", err)
	}

	// Clean up
	sched.StopTask(task.ID)
}

func TestScheduler_StartTask_AlreadyRunning(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	task, _ := sched.CreateTask(req, "/tmp/test.csv")

	err := sched.StartTask(task)
	if err != nil {
		t.Fatalf("first StartTask failed: %v", err)
	}

	err = sched.StartTask(task)
	if err != nil {
		t.Errorf("second StartTask should be no-op, got: %v", err)
	}

	sched.StopTask(task.ID)
}

func TestScheduler_StopTask_NotRunning(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	// Stopping a task that was never started should be a no-op
	err := sched.StopTask(uuid.New())
	if err != nil {
		t.Errorf("StopTask on unknown task should be no-op, got: %v", err)
	}
}

func TestScheduler_RecoverTasks(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test-recover.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	dbPath := tmpfile.Name()
	defer os.Remove(dbPath)

	sqlite1, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	req := makeCreateReq()
	taskID := uuid.New()
	err = sqlite1.CreateTask(&models.Task{
		ID:        taskID,
		Name:      req.Name,
		InputType: req.InputType,
		FilePath:  "/tmp/test.csv",
		SrcIP:     req.SrcIP,
		DstIP:     req.DstIP,
		SrcMAC:    req.SrcMAC,
		DstMAC:    req.DstMAC,
		QoS:       req.QoS,
		Status:    models.TaskStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		sqlite1.Close()
		t.Fatal(err)
	}
	sqlite1.Close()

	sqlite2, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite2.Close()

	redis2 := testutil.NewMockRedis()
	sched2 := NewTaskScheduler(sqlite2, redis2)

	status, err := sched2.GetTaskStatus(taskID)
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	if status != models.TaskStatusRunning {
		t.Errorf("recovered task status = %s, want running", status)
	}

	sched2.StopTask(taskID)
}

func TestScheduler_DurationMs(t *testing.T) {
	sched, cleanup := setupTestScheduler(t)
	defer cleanup()

	req := makeCreateReq()
	req.DurationMs = 30000

	task, err := sched.CreateTask(req, "/tmp/test.csv")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if task.DurationMs != 30000 {
		t.Errorf("DurationMs = %d, want 30000", task.DurationMs)
	}
}
