package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/service"
)

type RepoHandler struct {
	svc *service.RepoService
}

func NewRepoHandler(svc *service.RepoService) *RepoHandler {
	return &RepoHandler{svc: svc}
}

func (h *RepoHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	repos, err := h.svc.ListProducts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

func (h *RepoHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	product := chi.URLParam(r, "product")
	subPath := r.URL.Query().Get("path")
	relPath := product
	if subPath != "" {
		relPath = product + "/" + subPath
	}

	depth := 3
	if d := r.URL.Query().Get("depth"); d != "" {
		depth, _ = strconv.Atoi(d)
		if depth < 1 {
			depth = 1
		}
		if depth > 10 {
			depth = 10
		}
	}

	tree, err := h.svc.GetTree(relPath, depth)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *RepoHandler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	product := chi.URLParam(r, "product")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}
	relPath := product + "/" + filePath

	content, err := h.svc.GetFileContent(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func (h *RepoHandler) ListRollbacks(w http.ResponseWriter, r *http.Request) {
	product := chi.URLParam(r, "product")
	snapshots, err := h.svc.ListRollbacks(product)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshots)
}

func (h *RepoHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	product := chi.URLParam(r, "product")
	var req struct {
		Snapshot string `json:"snapshot"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Snapshot == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "snapshot required"})
		return
	}

	if err := h.svc.Rollback(product, req.Snapshot); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
