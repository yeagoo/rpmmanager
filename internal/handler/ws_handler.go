package handler

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"github.com/ivmm/rpmmanager/internal/auth"
	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/service"
)

type WSHandler struct {
	cfg      *config.Config
	buildSvc *service.BuildService
	authSvc  *auth.Service
}

func NewWSHandler(cfg *config.Config, buildSvc *service.BuildService, authSvc *auth.Service) *WSHandler {
	return &WSHandler{cfg: cfg, buildSvc: buildSvc, authSvc: authSvc}
}

func (h *WSHandler) BuildLog(w http.ResponseWriter, r *http.Request) {
	buildID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Build allowed origins for WebSocket
	allowedOrigins := []string{h.cfg.Server.BaseURL}
	if h.cfg.Server.BaseURL == "" || h.cfg.Server.BaseURL == "http://localhost:8080" {
		allowedOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: allowedOrigins,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()

	// Send existing log content
	build, err := h.buildSvc.GetByID(buildID)
	if err != nil {
		conn.Close(websocket.StatusInternalError, "build not found")
		return
	}

	if build.LogFile != "" {
		if content, err := os.ReadFile(build.LogFile); err == nil && len(content) > 0 {
			conn.Write(ctx, websocket.MessageText, content)
		}
	}

	// Subscribe to live updates
	logWriter := h.buildSvc.GetLogWriter(buildID)
	if logWriter == nil {
		// Build already finished, send completion message
		conn.Write(ctx, websocket.MessageText, []byte("\n--- Build finished ---\n"))
		return
	}

	subID := uuid.New().String()
	ch := logWriter.Subscribe(subID)
	defer logWriter.Unsubscribe(subID)

	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				// Channel closed = build finished
				conn.Write(ctx, websocket.MessageText, []byte("\n--- Build finished ---\n"))
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
		}
	}
}
