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
		log.Printf("ws upgrade error: %v", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			h.pushStats(conn, taskID)
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *WSHandler) pushStats(conn *websocket.Conn, taskID uuid.UUID) {
	stats, err := h.scheduler.GetStats(taskID)
	if err != nil {
		log.Printf("ws pushStats get error: %v", err)
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
		if err != nil {
			log.Printf("ws pushStatus get error: %v", err)
		}
		return
	}
	msg := models.WSMessage{Type: "status_change", Data: map[string]interface{}{"status": status}}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("ws write status error: %v", err)
	}
}
