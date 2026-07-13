package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var (
	ErrDomainInUse       = errors.New("domain in use")
	ErrInvalidDomainHost = errors.New("invalid domain host")
	ErrInvalidDomainType = errors.New("invalid domain type")
)

type DomainService struct {
	db    *gorm.DB
	cache *DomainCache
}

type DomainInput struct {
	Host      string `json:"host"`
	Type      string `json:"type"`
	Scheme    string `json:"scheme"`
	Remark    string `json:"remark"`
	CreatedBy uint64 `json:"-"`
}

func NewDomainService(db *gorm.DB, cache *DomainCache) *DomainService {
	return &DomainService{db: db, cache: cache}
}

func (s *DomainService) List(ctx context.Context) ([]model.Domain, error) {
	var domains []model.Domain
	return domains, s.db.WithContext(ctx).Order("id DESC").Find(&domains).Error
}

func (s *DomainService) Create(ctx context.Context, input DomainInput) (model.Domain, error) {
	domain, err := domainFromInput(input)
	if err != nil {
		return model.Domain{}, err
	}

	if err := s.db.WithContext(ctx).Create(&domain).Error; err != nil {
		if isDuplicateError(err) {
			return model.Domain{}, ErrCodeConflict
		}
		return model.Domain{}, err
	}
	return domain, s.cache.Refresh()
}

func (s *DomainService) Update(ctx context.Context, id uint64, input DomainInput) (model.Domain, error) {
	domain, err := s.Get(ctx, id)
	if err != nil {
		return model.Domain{}, err
	}

	updates := map[string]interface{}{}
	if input.Host != "" {
		host := normalizeHost(input.Host)
		if !validDomainHost(host) {
			return model.Domain{}, ErrInvalidDomainHost
		}
		updates["host"] = host
	}
	if input.Type != "" {
		if !validDomainType(input.Type) {
			return model.Domain{}, ErrInvalidDomainType
		}
		updates["type"] = input.Type
	}
	if input.Scheme != "" {
		if input.Scheme != "http" && input.Scheme != "https" {
			return model.Domain{}, ErrInvalidDomainType
		}
		updates["scheme"] = input.Scheme
	}
	if input.Remark != "" {
		updates["remark"] = input.Remark
	}

	if len(updates) == 0 {
		return domain, nil
	}
	if err := s.db.WithContext(ctx).Model(&domain).Updates(updates).Error; err != nil {
		if isDuplicateError(err) {
			return model.Domain{}, ErrCodeConflict
		}
		return model.Domain{}, err
	}
	if err := s.cache.Refresh(); err != nil {
		return model.Domain{}, err
	}
	return s.Get(ctx, id)
}

func (s *DomainService) Delete(ctx context.Context, id uint64) error {
	domain, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	var count int64
	err = s.db.WithContext(ctx).Model(&model.Link{}).
		Where("entry_domain_id = ? OR transit_domain_id = ? OR landing_domain_id = ?", id, id, id).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrDomainInUse
	}

	if err := s.db.WithContext(ctx).Delete(&domain).Error; err != nil {
		return err
	}
	return s.cache.Refresh()
}

func (s *DomainService) Get(ctx context.Context, id uint64) (model.Domain, error) {
	var domain model.Domain
	err := s.db.WithContext(ctx).First(&domain, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Domain{}, ErrDomainNotFound
	}
	return domain, err
}

func domainFromInput(input DomainInput) (model.Domain, error) {
	host := normalizeHost(input.Host)
	if !validDomainHost(host) {
		return model.Domain{}, ErrInvalidDomainHost
	}
	if !validDomainType(input.Type) {
		return model.Domain{}, ErrInvalidDomainType
	}

	scheme := input.Scheme
	if scheme == "" {
		scheme = "https"
	}
	if scheme != "http" && scheme != "https" {
		return model.Domain{}, ErrInvalidDomainType
	}

	var remark *string
	if strings.TrimSpace(input.Remark) != "" {
		remark = optionalString(input.Remark)
	}

	return model.Domain{
		Host:      host,
		Type:      input.Type,
		Scheme:    scheme,
		Remark:    remark,
		Status:    model.StatusActive,
		CreatedBy: input.CreatedBy,
	}, nil
}

func validDomainHost(host string) bool {
	if host == "" || strings.Contains(host, "/") || strings.Contains(host, "://") {
		return false
	}
	return strings.Contains(host, ".") || host == "localhost"
}

func validDomainType(domainType string) bool {
	switch domainType {
	case model.DomainTypeEntry, model.DomainTypeTransit, model.DomainTypeLanding:
		return true
	default:
		return false
	}
}
