package api

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"dns-sender/internal/scheduler"
	"dns-sender/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TaskHandler struct {
	scheduler *scheduler.TaskScheduler
	uploadDir string
	pcapDir   string
}

func NewTaskHandler(sched *scheduler.TaskScheduler, uploadDir, pcapDir string) *TaskHandler {
	return &TaskHandler{
		scheduler: sched,
		uploadDir: uploadDir,
		pcapDir:   pcapDir,
	}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RandomSrcIP && req.Interface == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interface is required when random_src_ip is enabled"})
		return
	}

	filePath := ""
	if req.InputType == models.InputTypePCAP && req.FilePath != "" {
		// PCAP server-side path: join with base dir and validate
		fullPath := filepath.Join(h.pcapDir, req.FilePath)
		fullPath = filepath.Clean(fullPath)
		if !strings.HasPrefix(fullPath, filepath.Clean(h.pcapDir)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path does not exist"})
			return
		}
		if !info.IsDir() && filepath.Ext(fullPath) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path must be a directory or a pcap file"})
			return
		}
		filePath = fullPath
	} else if req.FileContent != "" {
		data, err := base64.StdEncoding.DecodeString(req.FileContent)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file content"})
			return
		}

		taskID := uuid.New().String()
		ext := ".csv"
		if req.InputType == models.InputTypePCAP {
			ext = ".pcap"
		}
		filePath = filepath.Join(h.uploadDir, taskID+ext)

		if err := os.WriteFile(filePath, data, 0600); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
	}

	task, err := h.scheduler.CreateTask(&req, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	tasks := h.scheduler.ListTasks()
	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.scheduler.UpdateTask(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.scheduler.GetTask(id)
	if err == nil && task.FilePath != "" {
		os.Remove(task.FilePath)
	}

	if err := h.scheduler.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

func (h *TaskHandler) DownloadFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if task.FilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no file for this task"})
		return
	}

	c.FileAttachment(task.FilePath, filepath.Base(task.FilePath))
}

func (h *TaskHandler) StartTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if err := h.scheduler.StartTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task started"})
}

func (h *TaskHandler) StopTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	if err := h.scheduler.StopTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task stopped"})
}

func (h *TaskHandler) GetTaskStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	stats, err := h.scheduler.GetStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stats not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	status, err := h.scheduler.GetTaskStatus(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "status not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

func (h *TaskHandler) ListPCAPDirs(c *gin.Context) {
	subpath := c.Query("path")
	fullPath := filepath.Join(h.pcapDir, subpath)
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(h.pcapDir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
		return
	}

	dirs := []string{}
	files := []string{}
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		} else if strings.HasSuffix(strings.ToLower(e.Name()), ".pcap") ||
			strings.HasSuffix(strings.ToLower(e.Name()), ".pcapng") ||
			strings.HasSuffix(strings.ToLower(e.Name()), ".cap") {
			files = append(files, e.Name())
		}
	}

	relPath := strings.TrimPrefix(fullPath, filepath.Clean(h.pcapDir))
	relPath = strings.TrimPrefix(relPath, "/")

	c.JSON(http.StatusOK, gin.H{
		"dirs":         dirs,
		"files":        files,
		"current_path": relPath,
	})
}