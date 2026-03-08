package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/service"
)

type RepoRPMHandler struct {
	cfg        *config.Config
	repoRPMSvc *service.RepoRPMService
	productSvc *service.ProductService
}

func NewRepoRPMHandler(cfg *config.Config, repoRPMSvc *service.RepoRPMService, productSvc *service.ProductService) *RepoRPMHandler {
	return &RepoRPMHandler{cfg: cfg, repoRPMSvc: repoRPMSvc, productSvc: productSvc}
}

// Generate creates a new repo RPM for a product.
func (h *RepoRPMHandler) Generate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	product, err := h.productSvc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	// Parse optional overrides from request body
	var body struct {
		Distros []string `json:"distros"`
		Version string   `json:"version"`
	}
	decodeJSON(r, &body)

	// Build the request from product data
	distros := product.TargetDistros
	if len(body.Distros) > 0 {
		distros = body.Distros
	}

	version := "1.0"
	if body.Version != "" {
		version = body.Version
	}

	baseURL := product.BaseURL
	if baseURL == "" {
		baseURL = h.cfg.Server.BaseURL
	}

	var gpgKeyID int64
	if product.GPGKeyID != nil {
		gpgKeyID = *product.GPGKeyID
	}

	req := &service.RepoRPMRequest{
		ProductName: product.Name,
		DisplayName: product.DisplayName,
		BaseURL:     baseURL,
		GPGKeyID:    gpgKeyID,
		Distros:     distros,
		Maintainer:  product.Maintainer,
		Vendor:      product.Vendor,
		Homepage:    product.Homepage,
		Version:     version,
	}

	result, err := h.repoRPMSvc.Generate(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	result.DownloadURL = fmt.Sprintf("/api/products/%d/repo-rpm/download", id)
	writeJSON(w, http.StatusOK, result)
}

// GetLatest returns info about the latest repo RPM for a product.
func (h *RepoRPMHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	product, err := h.productSvc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	result, err := h.repoRPMSvc.GetLatest(product.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if result == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no repo RPM found, generate one first"})
		return
	}

	result.DownloadURL = fmt.Sprintf("/api/products/%d/repo-rpm/download", id)
	writeJSON(w, http.StatusOK, result)
}

// Download serves the latest repo RPM file for download.
func (h *RepoRPMHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	product, err := h.productSvc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	result, err := h.repoRPMSvc.GetLatest(product.Name)
	if err != nil || result == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no repo RPM found"})
		return
	}

	w.Header().Set("Content-Type", "application/x-rpm")
	w.Header().Set("Content-Disposition", "attachment; filename="+result.FileName)
	http.ServeFile(w, r, result.FilePath)
}

// PublicDownload serves a specific repo RPM file without authentication.
func (h *RepoRPMHandler) PublicDownload(w http.ResponseWriter, r *http.Request) {
	productName := chi.URLParam(r, "product")
	fileName := chi.URLParam(r, "filename")

	filePath, err := h.repoRPMSvc.GetFilePath(productName, fileName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/x-rpm")
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	http.ServeFile(w, r, filePath)
}
