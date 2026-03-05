package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/service"
)

type GPGHandler struct {
	svc *service.GPGService
}

func NewGPGHandler(svc *service.GPGService) *GPGHandler {
	return &GPGHandler{svc: svc}
}

func (h *GPGHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.svc.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list keys"})
		return
	}
	if keys == nil {
		keys = []models.GPGKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *GPGHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	key, err := h.svc.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *GPGHandler) Import(w http.ResponseWriter, r *http.Request) {
	var keyData []byte

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			KeyData string `json:"key_data"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		keyData = []byte(req.KeyData)
	} else if strings.HasPrefix(contentType, "multipart/") {
		file, _, err := r.FormFile("key_file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key_file required for multipart upload"})
			return
		}
		defer file.Close()
		var readErr error
		keyData, readErr = io.ReadAll(io.LimitReader(file, 1<<20)) // 1MB limit
		if readErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read key file"})
			return
		}
	} else {
		// Raw body
		var err error
		keyData, err = io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || len(keyData) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key data required"})
			return
		}
	}

	if len(keyData) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empty key data"})
		return
	}

	key, err := h.svc.ImportKey(keyData)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

func (h *GPGHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and email are required"})
		return
	}

	key, err := h.svc.GenerateKey(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

func (h *GPGHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete key"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *GPGHandler) Export(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	armor, err := h.svc.Export(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to export key"})
		return
	}
	w.Header().Set("Content-Type", "application/pgp-keys")
	w.Write([]byte(armor))
}

func (h *GPGHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.svc.SetDefault(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set default key"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
