package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/repository"
)

// Pipeline orchestrates the 4-stage build pipeline.
type Pipeline struct {
	cfg               *config.Config
	buildRepo         *repository.BuildRepo
	product           *models.Product
	build             *models.Build
	log               *LogWriter
	gpgFingerprint    string // GPG key fingerprint for signing (empty = skip signing)
	rollbackKeepCount int    // number of rollback snapshots to keep (0 = default 3)
}

// New creates a new Pipeline.
func New(cfg *config.Config, buildRepo *repository.BuildRepo, product *models.Product, build *models.Build, logWriter *LogWriter, gpgFingerprint string, rollbackKeepCount int) *Pipeline {
	return &Pipeline{
		cfg:               cfg,
		buildRepo:         buildRepo,
		product:           product,
		build:             build,
		log:               logWriter,
		gpgFingerprint:    gpgFingerprint,
		rollbackKeepCount: rollbackKeepCount,
	}
}

// Run executes the 4-stage pipeline: build → sign → publish → verify.
func (p *Pipeline) Run(ctx context.Context) error {
	// Mark build as started
	p.buildRepo.UpdateStarted(p.build.ID)

	// Create staging directory
	stagingDir := filepath.Join(p.cfg.Storage.RepoRoot, p.product.Name+".staging")
	os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		p.buildRepo.UpdateFinished(p.build.ID, models.BuildStatusFailed, err.Error(), 0, 0)
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	var rpmCount, symlinkCount int

	stages := []struct {
		name   string
		status models.BuildStatus
		fn     func() error
	}{
		{
			name:   "build",
			status: models.BuildStatusBuilding,
			fn: func() error {
				var err error
				rpmCount, symlinkCount, err = StageBuild(ctx, p.cfg, p.product, p.build.Version, stagingDir, p.log)
				return err
			},
		},
		{
			name:   "sign",
			status: models.BuildStatusSigning,
			fn: func() error {
				return StageSign(ctx, p.cfg, p.gpgFingerprint, p.cfg.GPG.HomeDir, stagingDir, p.log)
			},
		},
		{
			name:   "publish",
			status: models.BuildStatusPublishing,
			fn: func() error {
				return StagePublish(ctx, p.cfg.Storage.RepoRoot, p.product.Name, stagingDir, p.rollbackKeepCount, p.log)
			},
		},
		{
			name:   "verify",
			status: models.BuildStatusVerifying,
			fn: func() error {
				productDir := filepath.Join(p.cfg.Storage.RepoRoot, p.product.Name)
				return StageVerify(ctx, p.cfg, productDir, p.gpgFingerprint, p.log)
			},
		},
	}

	for _, stage := range stages {
		select {
		case <-ctx.Done():
			p.buildRepo.UpdateFinished(p.build.ID, models.BuildStatusCancelled, "cancelled", rpmCount, symlinkCount)
			return ctx.Err()
		default:
		}

		p.log.WriteStage(stage.name, "starting")
		p.buildRepo.UpdateStatus(p.build.ID, stage.status, stage.name)

		if err := stage.fn(); err != nil {
			p.log.WriteStage(stage.name, fmt.Sprintf("FAILED: %s", err))
			p.buildRepo.UpdateFinished(p.build.ID, models.BuildStatusFailed, err.Error(), rpmCount, symlinkCount)
			return fmt.Errorf("stage %s: %w", stage.name, err)
		}

		p.log.WriteStage(stage.name, "completed")
	}

	p.buildRepo.UpdateFinished(p.build.ID, models.BuildStatusSuccess, "", rpmCount, symlinkCount)
	p.log.WriteLog("Build completed successfully: %d RPMs, %d symlinks", rpmCount, symlinkCount)
	return nil
}
