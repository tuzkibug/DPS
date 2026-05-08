package main

import (
	"log"
	"os"

	taskhandler "dns-sender/internal/api/handler"
	"dns-sender/internal/scheduler"
	"dns-sender/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	sqlitePath := getEnv("SQLITE_PATH", "./data.db")
	uploadDir := getEnv("UPLOAD_DIR", "./uploads")
	port := getEnv("PORT", "8080")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("failed to create upload dir: %v", err)
	}

	sqliteStore, err := store.NewSQLiteStore(sqlitePath)
	if err != nil {
		log.Fatalf("failed to init sqlite: %v", err)
	}
	defer sqliteStore.Close()

	redisStore, err := store.NewRedisStore(redisAddr)
	if err != nil {
		log.Fatalf("failed to init redis: %v", err)
	}
	defer redisStore.Close()

	sched := scheduler.NewTaskScheduler(sqliteStore, redisStore)
	taskHandler := taskhandler.NewTaskHandler(sched, uploadDir)
	wsHandler := taskhandler.NewWSHandler(sched)

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/tasks", taskHandler.CreateTask)
		v1.GET("/tasks", taskHandler.ListTasks)
		v1.GET("/tasks/:id", taskHandler.GetTask)
		v1.PUT("/tasks/:id", taskHandler.UpdateTask)
		v1.DELETE("/tasks/:id", taskHandler.DeleteTask)
		v1.POST("/tasks/:id/start", taskHandler.StartTask)
		v1.POST("/tasks/:id/stop", taskHandler.StopTask)
		v1.GET("/tasks/:id/stats", taskHandler.GetTaskStats)
		v1.GET("/tasks/:id/file", taskHandler.DownloadFile)
		v1.GET("/tasks/:id/status", taskHandler.GetTaskStatus)
		v1.GET("/ws/tasks/:id", wsHandler.HandleTaskWS)
	}

	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}