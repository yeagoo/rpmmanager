package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/service"
)

type BuildHandler struct {
	cfg *config.Config
	svc *service.BuildService
}

func NewBuildHandler(cfg *config.Config, svc *service.BuildService) *BuildHandler {
	return &BuildHandler{cfg: cfg, svc: svc}
}

func (h *BuildHandler) List(w http.ResponseWriter, r *http.Request) {
	productID, _ := strconv.ParseInt(r.URL.Query().Get("product_id"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	builds, err := h.svc.List(productID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list builds"})
		return
	}
	if builds == nil {
		builds = []models.Build{}
	}
	writeJSON(w, http.StatusOK, builds)
}

func (h *BuildHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	build, err := h.svc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
		return
	}

	writeJSON(w, http.StatusOK, build)
}

func (h *BuildHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req models.TriggerBuildRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ProductID == 0 || req.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id and version are required"})
		return
	}

	build, err := h.svc.TriggerBuild(&req, "manual")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, build)
}

func (h *BuildHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.svc.CancelBuild(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// maxLogSize limits how much log data we serve to avoid unbounded memory usage.
const maxLogSize = 10 << 20 // 10 MiB

func (h *BuildHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	build, err := h.svc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
		return
	}

	if build.LogFile == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no log file"})
		return
	}

	// Validate log path is within build_logs directory
	absLogDir, _ := filepath.Abs(h.cfg.Storage.BuildLogs)
	absLogFile, _ := filepath.Abs(build.LogFile)
	if !strings.HasPrefix(absLogFile, absLogDir+string(filepath.Separator)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid log path"})
		return
	}

	f, err := os.Open(absLogFile)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "log file not found"})
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, io.LimitReader(f, maxLogSize))
}
