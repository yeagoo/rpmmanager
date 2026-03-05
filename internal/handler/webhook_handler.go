package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/service"
)

type WebhookHandler struct {
	buildSvc   *service.BuildService
	productSvc *service.ProductService
}

func NewWebhookHandler(buildSvc *service.BuildService, productSvc *service.ProductService) *WebhookHandler {
	return &WebhookHandler{buildSvc: buildSvc, productSvc: productSvc}
}

func (h *WebhookHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	productName := chi.URLParam(r, "product")

	var req struct {
		Version string `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "version required"})
		return
	}

	product, err := h.productSvc.GetByName(productName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	build, err := h.buildSvc.TriggerBuild(&models.TriggerBuildRequest{
		ProductID: product.ID,
		Version:   req.Version,
	}, "webhook")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, build)
}
