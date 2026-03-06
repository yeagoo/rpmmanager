package handler

import (
	"database/sql"
	"net/http"
)

type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type DashboardData struct {
	ProductCount   int              `json:"product_count"`
	BuildCount     int              `json:"build_count"`
	GPGKeyCount    int              `json:"gpg_key_count"`
	ActiveBuilds   int              `json:"active_builds"`
	RecentBuilds   []DashboardBuild `json:"recent_builds"`
	ProductSummary []ProductSummary `json:"product_summary"`
}

type DashboardBuild struct {
	ID                 int64  `json:"id"`
	ProductDisplayName string `json:"product_display_name"`
	Version            string `json:"version"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
}

type ProductSummary struct {
	ID              int64  `json:"id"`
	DisplayName     string `json:"display_name"`
	LatestVersion   string `json:"latest_version"`
	LastBuildStatus string `json:"last_build_status"`
	Enabled         bool   `json:"enabled"`
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	data := DashboardData{}

	if err := h.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&data.ProductCount); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query dashboard"})
		return
	}
	h.db.QueryRow("SELECT COUNT(*) FROM builds").Scan(&data.BuildCount)
	h.db.QueryRow("SELECT COUNT(*) FROM gpg_keys").Scan(&data.GPGKeyCount)
	h.db.QueryRow(`SELECT COUNT(*) FROM builds WHERE status IN ('pending','building','signing','publishing','verifying')`).Scan(&data.ActiveBuilds)

	// Recent builds
	rows, err := h.db.Query(`
		SELECT b.id, p.display_name, b.version, b.status, b.created_at
		FROM builds b JOIN products p ON p.id = b.product_id
		ORDER BY b.created_at DESC LIMIT 10`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var b DashboardBuild
			if err := rows.Scan(&b.ID, &b.ProductDisplayName, &b.Version, &b.Status, &b.CreatedAt); err != nil {
				continue
			}
			data.RecentBuilds = append(data.RecentBuilds, b)
		}
	}
	if data.RecentBuilds == nil {
		data.RecentBuilds = []DashboardBuild{}
	}

	// Product summary with latest build
	rows2, err := h.db.Query(`
		SELECT p.id, p.display_name, p.enabled,
			COALESCE((SELECT version FROM builds WHERE product_id = p.id ORDER BY created_at DESC LIMIT 1), ''),
			COALESCE((SELECT status FROM builds WHERE product_id = p.id ORDER BY created_at DESC LIMIT 1), '')
		FROM products p ORDER BY p.display_name`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var s ProductSummary
			if err := rows2.Scan(&s.ID, &s.DisplayName, &s.Enabled, &s.LatestVersion, &s.LastBuildStatus); err != nil {
				continue
			}
			data.ProductSummary = append(data.ProductSummary, s)
		}
	}
	if data.ProductSummary == nil {
		data.ProductSummary = []ProductSummary{}
	}

	writeJSON(w, http.StatusOK, data)
}
