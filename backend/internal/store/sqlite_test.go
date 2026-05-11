package store

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"dns-sender/pkg/models"

	_ "github.com/mattn/go-sqlite3"
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

func TestSchemaMigration_FreshInstall(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify all expected columns exist after fresh init
	rows, err := store.db.Query("PRAGMA table_info(tasks)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}

	expected := []string{"id", "name", "input_type", "file_path", "src_ip", "dst_ip",
		"src_mac", "dst_mac", "target_qps", "jitter", "delay_min_ms", "delay_max_ms",
		"status", "created_at", "updated_at", "last_run_at", "total_run_ms"}
	for _, col := range expected {
		if !columns[col] {
			t.Errorf("column %q missing after fresh install", col)
		}
	}
}

func TestSchemaMigration_Idempotent(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Running migration again should be safe (idempotent)
	if err := store.migrateSchema(); err != nil {
		t.Fatalf("Second migrateSchema failed: %v", err)
	}
	// Third time too
	if err := store.migrateSchema(); err != nil {
		t.Fatalf("Third migrateSchema failed: %v", err)
	}
}

func TestSchemaMigration_UpgradeFromOldSchema(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test-old.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create old schema without last_run_at and total_run_ms
	oldSchema := `
	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		input_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		src_ip TEXT NOT NULL,
		dst_ip TEXT NOT NULL,
		src_mac TEXT NOT NULL,
		dst_mac TEXT NOT NULL,
		start_time TEXT,
		duration_ms INTEGER DEFAULT 0,
		target_qps INTEGER DEFAULT 100,
		jitter REAL DEFAULT 0,
		delay_min_ms INTEGER DEFAULT 0,
		delay_max_ms INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`
	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatal(err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrateSchema(); err != nil {
		t.Fatalf("migrateSchema failed: %v", err)
	}

	// Verify columns were added
	rows, err := db.Query("PRAGMA table_info(tasks)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue *string
		var pk int
		rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		columns[name] = true
	}

	if !columns["last_run_at"] {
		t.Error("last_run_at column was not added by migration")
	}
	if !columns["total_run_ms"] {
		t.Error("total_run_ms column was not added by migration")
	}

	// Should be able to INSERT/query with new columns
	_, err = db.Exec(`INSERT INTO tasks (id, name, input_type, file_path, src_ip, dst_ip, src_mac, dst_mac,
		status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-id", "test", "csv", "/f.csv", "1.1.1.1", "2.2.2.2",
		"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66", "pending",
		time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("INSERT after migration failed: %v", err)
	}
}