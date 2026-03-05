package service

import (
	"fmt"
	"regexp"

	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/repository"
)

var nameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

type ProductService struct {
	repo *repository.ProductRepo
}

func NewProductService(repo *repository.ProductRepo) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(p *models.Product) (int64, error) {
	if err := s.validate(p); err != nil {
		return 0, err
	}
	if p.NfpmConfig == "" {
		p.NfpmConfig = "{}"
	}
	if p.ExtraFiles == "" {
		p.ExtraFiles = "[]"
	}
	if len(p.Architectures) == 0 {
		p.Architectures = []string{"x86_64", "aarch64"}
	}
	p.Enabled = true
	return s.repo.Create(p)
}

func (s *ProductService) GetByID(id int64) (*models.Product, error) {
	return s.repo.GetByID(id)
}

func (s *ProductService) GetByName(name string) (*models.Product, error) {
	return s.repo.GetByName(name)
}

func (s *ProductService) List() ([]models.ProductListItem, error) {
	return s.repo.List()
}

func (s *ProductService) Update(p *models.Product) error {
	if err := s.validate(p); err != nil {
		return err
	}
	return s.repo.Update(p)
}

func (s *ProductService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ProductService) Duplicate(id int64) (int64, error) {
	return s.repo.Duplicate(id)
}

func (s *ProductService) validate(p *models.Product) error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(p.Name) < 2 {
		return fmt.Errorf("name must be at least 2 characters")
	}
	if !nameRegex.MatchString(p.Name) {
		return fmt.Errorf("name must be lowercase alphanumeric with hyphens (e.g., caddy, lite-speed)")
	}
	if p.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if p.SourceType != "github" && p.SourceType != "url" {
		return fmt.Errorf("source_type must be 'github' or 'url'")
	}
	if p.SourceType == "github" {
		if p.SourceGithubOwner == "" || p.SourceGithubRepo == "" {
			return fmt.Errorf("github owner and repo are required for github source type")
		}
	}
	if p.SourceType == "url" && p.SourceURLTemplate == "" {
		return fmt.Errorf("url template is required for url source type")
	}
	return nil
}
