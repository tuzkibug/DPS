package api

import (
	"net/http"

	"dns-sender/internal/scheduler"
	"dns-sender/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSHandler struct {
	scheduler *scheduler.TaskScheduler
}

func NewWSHandler(sched *scheduler.TaskScheduler) *WSHandler {
	return &WSHandler{scheduler: sched}
}

func (h *WSHandler) HandleTaskWS(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	stats, err := h.scheduler.GetStats(taskID)
	if err == nil {
		msg := models.WSMessage{
			Type: "stats",
			Data: stats,
		}
		conn.WriteJSON(msg)
	}

	status, _ := h.scheduler.GetTaskStatus(taskID)
	if status != "" {
		msg := models.WSMessage{
			Type: "status_change",
			Data: map[string]interface{}{"status": status},
		}
		conn.WriteJSON(msg)
	}
}