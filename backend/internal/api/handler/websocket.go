package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"dns-sender/internal/scheduler"
	"dns-sender/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	scheduler *scheduler.TaskScheduler
	upgrader  websocket.Upgrader
}

func NewWSHandler(sched *scheduler.TaskScheduler, allowedOrigins []string) *WSHandler {
	return &WSHandler{
		scheduler: sched,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // allow non-browser clients
				}
				for _, allowed := range allowedOrigins {
					if strings.EqualFold(origin, allowed) {
						return true
					}
				}
				return false
			},
		},
	}
}

func (h *WSHandler) HandleTaskWS(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Push initial state
	h.pushStats(conn, taskID)
	h.pushStatus(conn, taskID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			h.pushStats(conn, taskID)
		}
	}
}

func (h *WSHandler) pushStats(conn *websocket.Conn, taskID uuid.UUID) {
	stats, err := h.scheduler.GetStats(taskID)
	if err != nil {
		return
	}
	msg := models.WSMessage{Type: "stats", Data: stats}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("ws write stats error: %v", err)
	}
}

func (h *WSHandler) pushStatus(conn *websocket.Conn, taskID uuid.UUID) {
	status, err := h.scheduler.GetTaskStatus(taskID)
	if err != nil || status == "" {
		return
	}
	msg := models.WSMessage{Type: "status_change", Data: map[string]interface{}{"status": status}}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("ws write status error: %v", err)
	}
}
