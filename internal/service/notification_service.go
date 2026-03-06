package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ivmm/rpmmanager/internal/repository"
)

// NotificationService sends webhook notifications on build events.
type NotificationService struct {
	settingsRepo *repository.SettingsRepo
	client       *http.Client
}

// BuildNotification contains the data sent in webhook notifications.
type BuildNotification struct {
	Event       string `json:"event"` // "build.success" or "build.failed"
	ProductName string `json:"product_name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	RPMCount    int    `json:"rpm_count"`
	Duration    string `json:"duration"`
	Error       string `json:"error,omitempty"`
	Timestamp   string `json:"timestamp"`
}

func NewNotificationService(settingsRepo *repository.SettingsRepo) *NotificationService {
	return &NotificationService{
		settingsRepo: settingsRepo,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// NotifyBuildComplete sends a notification after a build finishes.
func (s *NotificationService) NotifyBuildComplete(notification *BuildNotification) {
	url, err := s.settingsRepo.Get("notification_url")
	if err != nil || url == "" {
		return // No webhook configured
	}

	// Check if this event type is enabled
	events, _ := s.settingsRepo.Get("notification_events")
	if events == "" {
		events = "build.success,build.failed" // Default: notify on all events
	}
	if !strings.Contains(events, notification.Event) {
		return
	}

	notification.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Detect notification type and format accordingly
	var body []byte
	contentType := "application/json"

	switch {
	case strings.Contains(url, "api.telegram.org"):
		body = s.formatTelegram(notification)
	case strings.Contains(url, "qyapi.weixin.qq.com"):
		body = s.formatWeChatWork(notification)
	case strings.Contains(url, "oapi.dingtalk.com"):
		body = s.formatDingTalk(notification)
	default:
		body, _ = json.Marshal(notification)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create notification request")
		return
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send notification")
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Warn().Int("status", resp.StatusCode).Str("url", url).Msg("Notification webhook returned error")
	}
}

func (s *NotificationService) formatMessage(n *BuildNotification) string {
	icon := "✅"
	if n.Event == "build.failed" {
		icon = "❌"
	}

	msg := fmt.Sprintf("%s Build %s: %s v%s\nRPMs: %d | Duration: %s",
		icon, n.Status, n.ProductName, n.Version, n.RPMCount, n.Duration)

	if n.Error != "" {
		msg += fmt.Sprintf("\nError: %s", n.Error)
	}

	return msg
}

func (s *NotificationService) formatTelegram(n *BuildNotification) []byte {
	payload := map[string]interface{}{
		"text":       s.formatMessage(n),
		"parse_mode": "HTML",
	}
	body, _ := json.Marshal(payload)
	return body
}

func (s *NotificationService) formatWeChatWork(n *BuildNotification) []byte {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": s.formatMessage(n),
		},
	}
	body, _ := json.Marshal(payload)
	return body
}

func (s *NotificationService) formatDingTalk(n *BuildNotification) []byte {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": s.formatMessage(n),
		},
	}
	body, _ := json.Marshal(payload)
	return body
}
