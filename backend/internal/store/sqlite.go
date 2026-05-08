package store

import (
	"database/sql"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
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
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) CreateTask(task *models.Task) error {
	qos := task.QoS
	_, err := s.db.Exec(`
		INSERT INTO tasks (id, name, input_type, file_path, src_ip, dst_ip, src_mac, dst_mac,
			start_time, duration_ms, target_qps, jitter, delay_min_ms, delay_max_ms, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID.String(), task.Name, task.InputType, task.FilePath,
		task.SrcIP, task.DstIP, task.SrcMAC, task.DstMAC,
		task.StartTime.Format(time.RFC3339), task.DurationMs,
		qos.TargetQPS, qos.Jitter, qos.DelayMinMs, qos.DelayMaxMs,
		task.Status, task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) GetTask(id uuid.UUID) (*models.Task, error) {
	row := s.db.QueryRow(`
		SELECT id, name, input_type, file_path, src_ip, dst_ip, src_mac, dst_mac,
			start_time, duration_ms, target_qps, jitter, delay_min_ms, delay_max_ms, status, created_at, updated_at
		FROM tasks WHERE id = ?`, id.String())

	var task models.Task
	var startTimeStr, createdAtStr, updatedAtStr string
	err := row.Scan(
		&task.ID, &task.Name, &task.InputType, &task.FilePath,
		&task.SrcIP, &task.DstIP, &task.SrcMAC, &task.DstMAC,
		&startTimeStr, &task.DurationMs,
		&task.QoS.TargetQPS, &task.QoS.Jitter, &task.QoS.DelayMinMs, &task.QoS.DelayMaxMs,
		&task.Status, &createdAtStr, &updatedAtStr)

	if err != nil {
		return nil, err
	}

	if startTimeStr != "" {
		task.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	task.ID, _ = uuid.Parse(task.ID.String())
	return &task, nil
}

func (s *SQLiteStore) ListTasks() ([]*models.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, name, input_type, file_path, src_ip, dst_ip, src_mac, dst_mac,
			start_time, duration_ms, target_qps, jitter, delay_min_ms, delay_max_ms, status, created_at, updated_at
		FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*models.Task, 0)
	for rows.Next() {
		var task models.Task
		var startTimeStr, createdAtStr, updatedAtStr string
		err := rows.Scan(
			&task.ID, &task.Name, &task.InputType, &task.FilePath,
			&task.SrcIP, &task.DstIP, &task.SrcMAC, &task.DstMAC,
			&startTimeStr, &task.DurationMs,
			&task.QoS.TargetQPS, &task.QoS.Jitter, &task.QoS.DelayMinMs, &task.QoS.DelayMaxMs,
			&task.Status, &createdAtStr, &updatedAtStr)
		if err != nil {
			continue
		}
		if startTimeStr != "" {
			task.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
		}
		task.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		task.ID, _ = uuid.Parse(task.ID.String())
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *SQLiteStore) UpdateTask(task *models.Task) error {
	qos := task.QoS
	_, err := s.db.Exec(`
		UPDATE tasks SET name=?, src_ip=?, dst_ip=?, src_mac=?, dst_mac=?,
			start_time=?, duration_ms=?, target_qps=?, jitter=?, delay_min_ms=?, delay_max_ms=?,
			status=?, updated_at=? WHERE id=?`,
		task.Name, task.SrcIP, task.DstIP, task.SrcMAC, task.DstMAC,
		task.StartTime.Format(time.RFC3339), task.DurationMs,
		qos.TargetQPS, qos.Jitter, qos.DelayMinMs, qos.DelayMaxMs,
		task.Status, time.Now().Format(time.RFC3339), task.ID.String())
	return err
}

func (s *SQLiteStore) DeleteTask(id uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id.String())
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}