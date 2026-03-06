package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/distromap"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if items == nil {
		items = []models.ProductListItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	if err := decodeJSON(r, &p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	id, err := h.svc.Create(&p)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, _ := h.svc.GetByID(id)
	writeJSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	product, err := h.svc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var p models.Product
	if err := decodeJSON(r, &p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	p.ID = id

	if err := h.svc.Update(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, _ := h.svc.GetByID(id)
	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.svc.Delete(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ProductHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	newID, err := h.svc.Duplicate(id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, _ := h.svc.GetByID(newID)
	writeJSON(w, http.StatusCreated, product)
}

// GetDistros returns all available distros and product lines for the UI.
func (h *ProductHandler) GetDistros(w http.ResponseWriter, r *http.Request) {
	type distroInfo struct {
		ProductLines []distromap.ProductLine              `json:"product_lines"`
		DistroGroups map[string][]distromap.DistroVersion `json:"distro_groups"`
		AllDistros   []string                             `json:"all_distros"`
	}

	writeJSON(w, http.StatusOK, distroInfo{
		ProductLines: distromap.DefaultProductLines,
		DistroGroups: distromap.AllDistros(),
		AllDistros:   distromap.AllDistroList(),
	})
}

// Export exports a product definition as JSON for backup/migration.
func (h *ProductHandler) Export(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	product, err := h.svc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	// Clear server-specific fields for portability
	export := *product
	export.ID = 0
	export.GPGKeyID = nil
	export.CreatedAt = time.Time{}
	export.UpdatedAt = time.Time{}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal failed"})
		return
	}

	fileName := fmt.Sprintf("%s.json", product.Name)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Write(data)
}

// ExportAll exports all product definitions as a JSON array.
func (h *ProductHandler) ExportAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Fetch full product data for each item
	var products []models.Product
	for _, item := range items {
		product, err := h.svc.GetByID(item.ID)
		if err != nil {
			continue
		}
		export := *product
		export.ID = 0
		export.GPGKeyID = nil
		export.CreatedAt = time.Time{}
		export.UpdatedAt = time.Time{}
		products = append(products, export)
	}

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=products-export.json")
	w.Write(data)
}

// Import creates products from a JSON array of product definitions.
func (h *ProductHandler) Import(w http.ResponseWriter, r *http.Request) {
	var products []models.Product
	if err := decodeJSON(r, &products); err != nil {
		// Try single product
		var single models.Product
		r.Body.Close()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: expected array of products"})
		_ = single
		return
	}

	var imported []map[string]interface{}
	var errors []string

	for _, p := range products {
		p.ID = 0
		id, err := h.svc.Create(&p)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", p.Name, err.Error()))
			continue
		}
		imported = append(imported, map[string]interface{}{
			"id":   id,
			"name": p.Name,
		})
	}

	result := map[string]interface{}{
		"imported": imported,
		"count":    len(imported),
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	writeJSON(w, http.StatusOK, result)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
