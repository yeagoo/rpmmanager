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
	cfg       *config.Config
	buildRepo *repository.BuildRepo
	product   *models.Product
	build     *models.Build
	log       *LogWriter
}

// New creates a new Pipeline.
func New(cfg *config.Config, buildRepo *repository.BuildRepo, product *models.Product, build *models.Build, logWriter *LogWriter) *Pipeline {
	return &Pipeline{
		cfg:       cfg,
		buildRepo: buildRepo,
		product:   product,
		build:     build,
		log:       logWriter,
	}
}

// Run executes the 4-stage pipeline: build → sign → publish → verify.
func (p *Pipeline) Run(ctx context.Context) error {
	// Create staging directory
	stagingDir := filepath.Join(p.cfg.Storage.RepoRoot, p.product.Name+".staging")
	os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	// Mark build as started
	p.buildRepo.UpdateStarted(p.build.ID)

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
				gpgKeyID := "" // TODO: resolve from product's gpg_key_id
				return StageSign(ctx, p.cfg, gpgKeyID, p.cfg.GPG.HomeDir, stagingDir, p.log)
			},
		},
		{
			name:   "publish",
			status: models.BuildStatusPublishing,
			fn: func() error {
				return StagePublish(ctx, p.cfg.Storage.RepoRoot, p.product.Name, stagingDir, p.log)
			},
		},
		{
			name:   "verify",
			status: models.BuildStatusVerifying,
			fn: func() error {
				productDir := filepath.Join(p.cfg.Storage.RepoRoot, p.product.Name)
				gpgKeyID := "" // TODO: resolve from product's gpg_key_id
				return StageVerify(ctx, p.cfg, productDir, gpgKeyID, p.log)
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
