package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/service"
)

type MonitorHandler struct {
	svc *service.MonitorService
}

func NewMonitorHandler(svc *service.MonitorService) *MonitorHandler {
	return &MonitorHandler{svc: svc}
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.svc.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, monitors)
}

func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}
	monitor, err := h.svc.GetByProductID(productID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}
	var req models.UpdateMonitorRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.svc.Update(productID, &req); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update monitor"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MonitorHandler) CheckNow(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
		return
	}
	version, err := h.svc.CheckNow(productID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": *version})
}
