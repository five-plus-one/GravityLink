package service

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var (
	ErrDomainInUse        = errors.New("domain in use")
	ErrInvalidDomainHost  = errors.New("invalid domain host")
	ErrInvalidDomainType  = errors.New("invalid domain type")
	ErrInvalidHomeMode    = errors.New("invalid home mode")
	ErrInvalidHomeLanding = errors.New("invalid home landing page")
)

type DomainService struct {
	db    *gorm.DB
	cache *DomainCache
}

type DomainInput struct {
	Host              string  `json:"host"`
	Type              string  `json:"type"`
	Scheme            string  `json:"scheme"`
	Remark            string  `json:"remark"`
	HomeMode          string  `json:"home_mode"`
	HomeRedirectURL   string  `json:"home_redirect_url"`
	HomeLandingPageID *uint64 `json:"home_landing_page_id"`
	CreatedBy         uint64  `json:"-"`
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
	if domain.HomeMode == model.HomeModeLanding {
		checked, err := s.applyHomeConfig(ctx, domain, input)
		if err != nil {
			return model.Domain{}, err
		}
		domain.HomeMode = checked.HomeMode
		domain.HomeRedirectURL = checked.HomeRedirectURL
		domain.HomeLandingPageID = checked.HomeLandingPageID
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
		host := strings.ToLower(strings.TrimSpace(input.Host))
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
	if input.HomeMode != "" || input.HomeRedirectURL != "" || input.HomeLandingPageID != nil {
		home, err := s.applyHomeConfig(ctx, domain, input)
		if err != nil {
			return model.Domain{}, err
		}
		updates["home_mode"] = home.HomeMode
		updates["home_redirect_url"] = home.HomeRedirectURL
		updates["home_landing_page_id"] = home.HomeLandingPageID
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
	host := strings.ToLower(strings.TrimSpace(input.Host))
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

	homeMode := input.HomeMode
	if homeMode == "" {
		homeMode = model.HomeModeDefault
	}
	if !validHomeMode(homeMode) {
		return model.Domain{}, ErrInvalidHomeMode
	}
	var homeURL *string
	if u := sanitizeHomeRedirectURL(input.HomeRedirectURL); u != "" {
		homeURL = &u
	}

	return model.Domain{
		Host:              host,
		Type:              input.Type,
		Scheme:            scheme,
		Remark:            remark,
		HomeMode:          homeMode,
		HomeRedirectURL:   homeURL,
		HomeLandingPageID: input.HomeLandingPageID,
		Status:            model.StatusActive,
		CreatedBy:         input.CreatedBy,
	}, nil
}

// applyHomeConfig 校验并归一化首页配置（Create/Update 共用）。
func (s *DomainService) applyHomeConfig(ctx context.Context, current model.Domain, input DomainInput) (model.Domain, error) {
	mode := input.HomeMode
	if mode == "" {
		mode = current.HomeMode
	}
	if mode == "" {
		mode = model.HomeModeDefault
	}
	if !validHomeMode(mode) {
		return model.Domain{}, ErrInvalidHomeMode
	}

	var homeURL *string
	if u := sanitizeHomeRedirectURL(input.HomeRedirectURL); u != "" {
		homeURL = &u
	}

	var landingID *uint64
	if mode == model.HomeModeLanding {
		if input.HomeLandingPageID == nil || *input.HomeLandingPageID == 0 {
			return model.Domain{}, ErrInvalidHomeLanding
		}
		var page model.LandingPage
		if err := s.db.WithContext(ctx).First(&page, *input.HomeLandingPageID).Error; err != nil {
			return model.Domain{}, ErrInvalidHomeLanding
		}
		// 活码/客服/卡密依赖具体链接上下文，不能直接作域名首页。
		if page.Template != "custom" && page.Template != "redirect_notice" {
			return model.Domain{}, ErrInvalidHomeLanding
		}
		id := page.ID
		landingID = &id
	}

	current.HomeMode = mode
	current.HomeRedirectURL = homeURL
	current.HomeLandingPageID = landingID
	return current, nil
}

func validHomeMode(mode string) bool {
	switch mode {
	case model.HomeModeDefault, model.HomeModeRedirect, model.HomeModeLanding:
		return true
	default:
		return false
	}
}

func validDomainHost(host string) bool {
	if host == "" || strings.ContainsAny(host, "/?#@ \\") {
		return false
	}
	u, err := url.Parse("http://" + host)
	if err != nil || u.Hostname() == "" {
		return false
	}
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err != nil || p < 1 || p > 65535 {
			return false
		}
	}
	return strings.Contains(u.Hostname(), ".") || u.Hostname() == "localhost" || strings.Contains(u.Hostname(), ":")
}

func validDomainType(domainType string) bool {
	switch domainType {
	case model.DomainTypeEntry, model.DomainTypeTransit, model.DomainTypeLanding:
		return true
	default:
		return false
	}
}
