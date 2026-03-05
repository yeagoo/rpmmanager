package handler

import (
	"net/http"
	"strconv"

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
		ProductLines []distromap.ProductLine            `json:"product_lines"`
		DistroGroups map[string][]distromap.DistroVersion `json:"distro_groups"`
		AllDistros   []string                           `json:"all_distros"`
	}

	writeJSON(w, http.StatusOK, distroInfo{
		ProductLines: distromap.DefaultProductLines,
		DistroGroups: distromap.AllDistros(),
		AllDistros:   distromap.AllDistroList(),
	})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
