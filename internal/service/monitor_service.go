package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/repository"
)

type MonitorService struct {
	cfg          *config.Config
	repo         *repository.MonitorRepo
	settingsRepo *repository.SettingsRepo
	buildSvc     *BuildService
	stopCh       chan struct{}
	mu           sync.Mutex
}

func NewMonitorService(cfg *config.Config, repo *repository.MonitorRepo, settingsRepo *repository.SettingsRepo, buildSvc *BuildService) *MonitorService {
	return &MonitorService{
		cfg:          cfg,
		repo:         repo,
		settingsRepo: settingsRepo,
		buildSvc:     buildSvc,
		stopCh:       make(chan struct{}),
	}
}

func (s *MonitorService) List() ([]models.Monitor, error) {
	monitors, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	if monitors == nil {
		monitors = []models.Monitor{}
	}
	return monitors, nil
}

func (s *MonitorService) GetByProductID(productID int64) (*models.Monitor, error) {
	return s.repo.GetByProductID(productID)
}

func (s *MonitorService) EnsureMonitor(productID int64) error {
	return s.repo.Upsert(productID)
}

func (s *MonitorService) Update(productID int64, req *models.UpdateMonitorRequest) error {
	// Ensure monitor exists
	s.repo.Upsert(productID)
	return s.repo.Update(productID, req.Enabled, req.CheckInterval, req.AutoBuild)
}

func (s *MonitorService) CheckNow(productID int64) (*string, error) {
	monitor, err := s.repo.GetByProductID(productID)
	if err != nil {
		return nil, err
	}

	version, err := s.checkLatestVersion(monitor)
	if err != nil {
		s.repo.UpdateCheckResult(productID, monitor.LastKnownVersion, err.Error())
		return nil, err
	}

	s.repo.UpdateCheckResult(productID, version, "")

	// If new version and auto-build enabled
	if version != monitor.LastKnownVersion && monitor.AutoBuild && version != "" {
		s.buildSvc.TriggerBuild(&models.TriggerBuildRequest{
			ProductID: productID,
			Version:   version,
		}, "monitor")
	}

	return &version, nil
}

// Start begins the background monitoring loop.
func (s *MonitorService) Start() {
	if !s.cfg.Monitor.Enabled {
		return
	}
	go s.loop()
}

func (s *MonitorService) Stop() {
	close(s.stopCh)
}

func (s *MonitorService) loop() {
	// Check every minute, individual monitors have their own intervals
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAll()
		}
	}
}

func (s *MonitorService) checkAll() {
	monitors, err := s.repo.GetEnabled()
	if err != nil {
		log.Printf("monitor: list enabled: %v", err)
		return
	}

	for _, m := range monitors {
		if !s.shouldCheck(m) {
			continue
		}
		s.CheckNow(m.ProductID)
	}
}

func (s *MonitorService) shouldCheck(m models.Monitor) bool {
	if m.LastCheckedAt == nil {
		return true
	}
	interval := parseDuration(m.CheckInterval)
	return time.Since(*m.LastCheckedAt) >= interval
}

func (s *MonitorService) checkLatestVersion(m *models.Monitor) (string, error) {
	if m.SourceType == "github" && m.SourceGithubOwner != "" && m.SourceGithubRepo != "" {
		return s.checkGitHubRelease(m.SourceGithubOwner, m.SourceGithubRepo)
	}
	return "", fmt.Errorf("unsupported source type: %s", m.SourceType)
}

func (s *MonitorService) checkGitHubRelease(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// Prefer settings DB token, fall back to config file
	token := s.cfg.Monitor.GithubToken
	if s.settingsRepo != nil {
		if dbToken, err := s.settingsRepo.Get("github_token"); err == nil && dbToken != "" {
			token = dbToken
		}
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github api: status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("github api decode: %w", err)
	}

	// Strip common tag prefixes (e.g. "v1.2.3", "release-1.2.3")
	version := release.TagName
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "release-")
	return version, nil
}

func parseDuration(s string) time.Duration {
	if strings.HasSuffix(s, "h") {
		s = strings.TrimSuffix(s, "h")
		var hours int
		fmt.Sscanf(s, "%d", &hours)
		return time.Duration(hours) * time.Hour
	}
	if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
		var mins int
		fmt.Sscanf(s, "%d", &mins)
		return time.Duration(mins) * time.Minute
	}
	// Default to 6 hours
	return 6 * time.Hour
}
