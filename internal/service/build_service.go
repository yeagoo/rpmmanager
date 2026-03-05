package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/pipeline"
	"github.com/ivmm/rpmmanager/internal/repository"
)

type BuildService struct {
	cfg         *config.Config
	buildRepo   *repository.BuildRepo
	productRepo *repository.ProductRepo
	mu          sync.Mutex
	running     map[int64]context.CancelFunc // productID -> cancel function
	logWriters  map[int64]*pipeline.LogWriter // buildID -> log writer
	writerMu    sync.RWMutex
}

func NewBuildService(cfg *config.Config, buildRepo *repository.BuildRepo, productRepo *repository.ProductRepo) *BuildService {
	return &BuildService{
		cfg:         cfg,
		buildRepo:   buildRepo,
		productRepo: productRepo,
		running:     make(map[int64]context.CancelFunc),
		logWriters:  make(map[int64]*pipeline.LogWriter),
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

		p := pipeline.New(s.cfg, s.buildRepo, product, build, logWriter)
		if err := p.Run(ctx); err != nil {
			log.Error().Err(err).Int64("build_id", buildID).Msg("Build failed")
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
