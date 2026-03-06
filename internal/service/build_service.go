package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/pipeline"
	"github.com/ivmm/rpmmanager/internal/repository"
)

type BuildService struct {
	cfg             *config.Config
	buildRepo       *repository.BuildRepo
	productRepo     *repository.ProductRepo
	gpgKeyRepo      *repository.GPGKeyRepo
	settingsRepo    *repository.SettingsRepo
	notificationSvc *NotificationService
	mu              sync.Mutex
	running         map[int64]context.CancelFunc  // productID -> cancel function
	logWriters      map[int64]*pipeline.LogWriter // buildID -> log writer
	writerMu        sync.RWMutex
}

func NewBuildService(cfg *config.Config, buildRepo *repository.BuildRepo, productRepo *repository.ProductRepo, gpgKeyRepo *repository.GPGKeyRepo, settingsRepo *repository.SettingsRepo, notificationSvc *NotificationService) *BuildService {
	return &BuildService{
		cfg:             cfg,
		buildRepo:       buildRepo,
		productRepo:     productRepo,
		gpgKeyRepo:      gpgKeyRepo,
		settingsRepo:    settingsRepo,
		notificationSvc: notificationSvc,
		running:         make(map[int64]context.CancelFunc),
		logWriters:      make(map[int64]*pipeline.LogWriter),
	}
}

func (s *BuildService) TriggerBuild(req *models.TriggerBuildRequest, triggerType string) (*models.Build, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing running build
	if _, ok := s.running[req.ProductID]; ok {
		return nil, fmt.Errorf("a build is already running for this product")
	}

	running, err := s.buildRepo.HasRunningBuild(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("check running builds: %w", err)
	}
	if running {
		return nil, fmt.Errorf("a build is already running for this product")
	}

	// Get product
	product, err := s.productRepo.GetByID(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	// Create build record
	logFileName := fmt.Sprintf("build-%d-%s.log", req.ProductID, uuid.New().String()[:8])
	logFilePath := filepath.Join(s.cfg.Storage.BuildLogs, logFileName)

	build := &models.Build{
		ProductID:     req.ProductID,
		Version:       req.Version,
		Status:        models.BuildStatusPending,
		TriggerType:   triggerType,
		TargetDistros: product.TargetDistros,
		Architectures: product.Architectures,
		LogFile:       logFilePath,
	}

	buildID, err := s.buildRepo.Create(build)
	if err != nil {
		return nil, fmt.Errorf("create build: %w", err)
	}
	build.ID = buildID

	// Create log writer
	logWriter, err := pipeline.NewLogWriter(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("create log writer: %w", err)
	}

	s.writerMu.Lock()
	s.logWriters[buildID] = logWriter
	s.writerMu.Unlock()

	// Start pipeline in background
	ctx, cancel := context.WithCancel(context.Background())
	s.running[req.ProductID] = cancel

	// Resolve GPG key fingerprint
	var gpgFingerprint string
	if product.GPGKeyID != nil {
		gpgKey, gpgErr := s.gpgKeyRepo.GetByID(*product.GPGKeyID)
		if gpgErr != nil {
			log.Warn().Err(gpgErr).Int64("gpg_key_id", *product.GPGKeyID).Msg("Failed to resolve GPG key, builds will not be signed")
		} else {
			gpgFingerprint = gpgKey.Fingerprint
		}
	}

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, req.ProductID)
			s.mu.Unlock()

			s.writerMu.Lock()
			delete(s.logWriters, buildID)
			s.writerMu.Unlock()

			logWriter.Close()
		}()

		startTime := time.Now()
		rollbackKeep := 3
		if s.settingsRepo != nil {
			if v, err := s.settingsRepo.Get("rollback_keep_count"); err == nil && v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					rollbackKeep = n
				}
			}
		}
		p := pipeline.New(s.cfg, s.buildRepo, product, build, logWriter, gpgFingerprint, rollbackKeep)
		runErr := p.Run(ctx)
		duration := time.Since(startTime)

		cancelled := ctx.Err() != nil
		if runErr != nil && !cancelled {
			log.Error().Err(runErr).Int64("build_id", buildID).Msg("Build failed")
		}

		// Send notification (skip for cancelled builds)
		if s.notificationSvc != nil && !cancelled {
			event := "build.success"
			status := "success"
			errMsg := ""
			if runErr != nil {
				event = "build.failed"
				status = "failed"
				errMsg = runErr.Error()
			}
			// Fetch updated build to get RPM count
			updatedBuild, _ := s.buildRepo.GetByID(buildID)
			rpmCount := 0
			if updatedBuild != nil {
				rpmCount = updatedBuild.RPMCount
			}
			s.notificationSvc.NotifyBuildComplete(&BuildNotification{
				Event:       event,
				ProductName: product.Name,
				Version:     build.Version,
				Status:      status,
				RPMCount:    rpmCount,
				Duration:    duration.Round(time.Second).String(),
				Error:       errMsg,
			})
		}
	}()

	return build, nil
}

func (s *BuildService) CancelBuild(buildID int64) error {
	build, err := s.buildRepo.GetByID(buildID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	cancel, ok := s.running[build.ProductID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("build is not running")
	}

	cancel()
	return nil
}

func (s *BuildService) GetByID(id int64) (*models.Build, error) {
	return s.buildRepo.GetByID(id)
}

func (s *BuildService) List(productID int64, limit int) ([]models.Build, error) {
	return s.buildRepo.List(productID, limit)
}

func (s *BuildService) GetLogWriter(buildID int64) *pipeline.LogWriter {
	s.writerMu.RLock()
	defer s.writerMu.RUnlock()
	return s.logWriters[buildID]
}
