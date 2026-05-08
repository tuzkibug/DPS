package store

import (
	"os"
	"testing"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	tmpfile, err := os.CreateTemp("", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	store, err := NewSQLiteStore(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatal(err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpfile.Name())
	}
	return store, cleanup
}

func TestSQLiteStore_CreateAndGetTask(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	task := &models.Task{
		ID:         uuid.New(),
		Name:       "Test Task",
		InputType:  models.InputTypeCSV,
		FilePath:   "/path/to/file.csv",
		SrcIP:      "192.168.1.100",
		DstIP:      "8.8.8.8",
		SrcMAC:     "aa:bb:cc:dd:ee:ff",
		DstMAC:     "11:22:33:44:55:66",
		StartTime:  time.Now(),
		DurationMs: 60000,
		QoS: models.QoSConfig{
			TargetQPS:  100,
			Jitter:     0.1,
			DelayMinMs: 0,
			DelayMaxMs: 100,
		},
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.CreateTask(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	retrieved, err := store.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.Name != task.Name {
		t.Errorf("Name = %q, want %q", retrieved.Name, task.Name)
	}
	if retrieved.SrcIP != task.SrcIP {
		t.Errorf("SrcIP = %q, want %q", retrieved.SrcIP, task.SrcIP)
	}
	if retrieved.QoS.TargetQPS != task.QoS.TargetQPS {
		t.Errorf("QoS.TargetQPS = %d, want %d", retrieved.QoS.TargetQPS, task.QoS.TargetQPS)
	}
}

func TestSQLiteStore_ListTasks(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		task := &models.Task{
			ID:         uuid.New(),
			Name:       "Task",
			InputType:  models.InputTypeCSV,
			FilePath:   "/path/to/file.csv",
			SrcIP:      "192.168.1.100",
			DstIP:      "8.8.8.8",
			SrcMAC:     "aa:bb:cc:dd:ee:ff",
			DstMAC:     "11:22:33:44:55:66",
			Status:     models.TaskStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		store.CreateTask(task)
	}

	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("ListTasks() returned %d tasks, want 3", len(tasks))
	}
}

func TestSQLiteStore_UpdateTask(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	task := &models.Task{
		ID:         uuid.New(),
		Name:       "Original Name",
		InputType:  models.InputTypeCSV,
		FilePath:   "/path/to/file.csv",
		SrcIP:      "192.168.1.100",
		DstIP:      "8.8.8.8",
		SrcMAC:     "aa:bb:cc:dd:ee:ff",
		DstMAC:     "11:22:33:44:55:66",
		Status:     models.TaskStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	store.CreateTask(task)

	task.Name = "Updated Name"
	task.Status = models.TaskStatusRunning

	err := store.UpdateTask(task)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	retrieved, _ := store.GetTask(task.ID)
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Updated Name")
	}
	if retrieved.Status != models.TaskStatusRunning {
		t.Errorf("Status = %q, want %q", retrieved.Status, models.TaskStatusRunning)
	}
}

func TestSQLiteStore_DeleteTask(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	task := &models.Task{
		ID:         uuid.New(),
		Name:       "To Delete",
		InputType:  models.InputTypeCSV,
		FilePath:   "/path/to/file.csv",
		SrcIP:      "192.168.1.100",
		DstIP:      "8.8.8.8",
		SrcMAC:     "aa:bb:cc:dd:ee:ff",
		DstMAC:     "11:22:33:44:55:66",
		Status:     models.TaskStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	store.CreateTask(task)

	err := store.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	_, err = store.GetTask(task.ID)
	if err == nil {
		t.Error("Expected error when getting deleted task, got nil")
	}
}

func TestSQLiteStore_GetNonExistentTask(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.GetTask(uuid.New())
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}