package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gravitylink/backend/internal/cache"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/pkg/base62"
)

var (
	ErrLinkNotFound      = errors.New("link not found")
	ErrLinkDisabled      = errors.New("link disabled")
	ErrLinkExpired       = errors.New("link expired")
	ErrInvalidCode       = errors.New("invalid code")
	ErrCodeConflict      = errors.New("code already exists")
	ErrInvalidTargetURL  = errors.New("invalid target url")
	ErrUnsupportedLink   = errors.New("unsupported link type")
	ErrTargetUnavailable = errors.New("target unavailable")
)

var customCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{2,32}$`)

type LinkService struct {
	db      *gorm.DB
	cache   *cache.LinkCache
	routing *RoutingService
}

type CreateLinkInput struct {
	Type            string         `json:"type"`
	Code            string         `json:"code"`
	EntryDomainID   uint64         `json:"entry_domain_id"`
	TransitDomainID *uint64        `json:"transit_domain_id"`
	LandingDomainID *uint64        `json:"landing_domain_id"`
	TargetURL       string         `json:"target_url"`
	LandingPageID   *uint64        `json:"landing_page_id"`
	Title           string         `json:"title"`
	AccessRule      string         `json:"access_rule"`
	OnlineSchedule  *string        `json:"online_schedule"`
	ExpireAt        *time.Time     `json:"expire_at"`
	CreatedBy       uint64         `json:"-"`
	Channel         *ChannelInput  `json:"channel"`
	Strategy        *StrategyInput `json:"strategy"`
}

type ChannelInput struct {
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMTerm     string `json:"utm_term"`
	UTMContent  string `json:"utm_content"`
}

type UpdateLinkInput struct {
	TargetURL      *string    `json:"target_url"`
	Title          *string    `json:"title"`
	ExpireAt       *time.Time `json:"expire_at"`
	Status         *string    `json:"status"`
	OnlineSchedule *string    `json:"online_schedule"`
	Code           *string    `json:"code"`
	EntryDomainID  *uint64    `json:"entry_domain_id"`
}

type ResolveResult struct {
	Link      cache.CachedLink
	TargetURL string
	Status    int
}

func NewLinkService(db *gorm.DB, linkCache *cache.LinkCache, routing *RoutingService) *LinkService {
	return &LinkService{db: db, cache: linkCache, routing: routing}
}

func (s *LinkService) Create(ctx context.Context, input CreateLinkInput) (model.Link, error) {
	linkType := input.Type
	if linkType == "" {
		linkType = model.LinkTypeShort
	}
	if linkType != model.LinkTypeShort && linkType != model.LinkTypeChannel && linkType != model.LinkTypeLiveQR {
		return model.Link{}, ErrUnsupportedLink
	}
	if linkType != model.LinkTypeLiveQR {
		if err := validateTargetURL(input.TargetURL); err != nil {
			return model.Link{}, err
		}
	}
	if linkType == model.LinkTypeLiveQR && (input.LandingPageID == nil || input.LandingDomainID == nil) {
		return model.Link{}, ErrTargetUnavailable
	}
	if input.Code != "" && !customCodePattern.MatchString(input.Code) {
		return model.Link{}, ErrInvalidCode
	}
	var entry model.Domain
	if err := s.db.WithContext(ctx).Where("id = ? AND type = ? AND status = ?", input.EntryDomainID, model.DomainTypeEntry, model.StatusActive).First(&entry).Error; err != nil {
		return model.Link{}, ErrTargetUnavailable
	}
	if linkType == model.LinkTypeLiveQR {
		var page model.LandingPage
		if err := s.db.WithContext(ctx).Where("id = ? AND domain_id = ? AND template IN ?", *input.LandingPageID, *input.LandingDomainID, []string{"liveqr", "kf", "kami"}).First(&page).Error; err != nil {
			return model.Link{}, ErrTargetUnavailable
		}
		var domain model.Domain
		if err := s.db.WithContext(ctx).Where("id = ? AND type = ? AND status = ?", *input.LandingDomainID, model.DomainTypeLanding, model.StatusActive).First(&domain).Error; err != nil {
			return model.Link{}, ErrTargetUnavailable
		}
		// kami 提取页无需轮换目标；kf/liveqr 需要二维码目标
		if page.Template != "kami" && (input.Strategy == nil || len(input.Strategy.Targets) == 0) {
			return model.Link{}, ErrNoRoutingTarget
		}
	}

	targetURL := input.TargetURL
	var title *string
	if input.Title != "" {
		title = &input.Title
	}
	var targetURLPtr *string
	if targetURL != "" {
		targetURLPtr = &targetURL
	}

	accessRule := input.AccessRule
	if accessRule != "wechat" && accessRule != "ios" && accessRule != "android" && accessRule != "mobile" && accessRule != "pc" {
		accessRule = "none"
	}

	link := model.Link{
		Code:            input.Code,
		Type:            linkType,
		EntryDomainID:   input.EntryDomainID,
		TransitDomainID: input.TransitDomainID,
		LandingDomainID: input.LandingDomainID,
		TargetURL:       targetURLPtr,
		LandingPageID:   input.LandingPageID,
		Title:           title,
		AccessRule:      accessRule,
		OnlineSchedule:  input.OnlineSchedule,
		ExpireAt:        input.ExpireAt,
		Status:          model.LinkStatusActive,
		CreatedBy:       input.CreatedBy,
	}

	if link.Code != "" {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
			if err := createChannelConfig(tx, link.ID, input.Channel, linkType); err != nil {
				return err
			}
			return s.createRoutingStrategy(tx, link.ID, input.Strategy, linkType)
		})
		if err != nil {
			return model.Link{}, normalizeCreateError(err)
		}
		return s.Get(ctx, link.ID)
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		link.Code = fmt.Sprintf("_pending_%d", time.Now().UnixNano())
		if len(link.Code) > 32 {
			link.Code = link.Code[:32]
		}
		if err := tx.Create(&link).Error; err != nil {
			return err
		}

		link.Code = base62.Encode(link.ID)
		for attempt := 0; attempt < 5; attempt++ {
			err := tx.Model(&link).Update("code", link.Code).Error
			if err == nil {
				break
			}
			if !errors.Is(normalizeCreateError(err), ErrCodeConflict) || attempt == 4 {
				return err
			}
			// Existing MySQL installations may have case-insensitive indexes.
			// Keep their schema and all existing short URLs unchanged.
			candidate := make([]byte, 6)
			if _, err = rand.Read(candidate); err != nil {
				return err
			}
			link.Code = hex.EncodeToString(candidate)
		}
		if err := createChannelConfig(tx, link.ID, input.Channel, linkType); err != nil {
			return err
		}
		return s.createRoutingStrategy(tx, link.ID, input.Strategy, linkType)
	})
	if err != nil {
		return model.Link{}, normalizeCreateError(err)
	}

	return s.Get(ctx, link.ID)
}

func (s *LinkService) List(ctx context.Context, linkType string) ([]model.Link, error) {
	var links []model.Link
	query := s.db.WithContext(ctx).Order("id DESC")
	if linkType != "" {
		query = query.Where("type = ?", linkType)
	}
	if err := query.Find(&links).Error; err != nil {
		return nil, err
	}
	var domains []model.Domain
	if err := s.db.WithContext(ctx).Where("type = ? AND status = ?", model.DomainTypeEntry, model.StatusActive).Find(&domains).Error; err != nil {
		return nil, err
	}
	for i := range links {
		for _, d := range domains {
			if d.ID == links[i].EntryDomainID {
				links[i].PublicURL = d.Scheme + "://" + d.Host + "/" + url.PathEscape(links[i].Code)
				break
			}
		}
	}
	return links, nil
}

func (s *LinkService) Get(ctx context.Context, id uint64) (model.Link, error) {
	var link model.Link
	err := s.db.WithContext(ctx).First(&link, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Link{}, ErrLinkNotFound
	}
	if err == nil {
		var d model.Domain
		if s.db.WithContext(ctx).Where("id = ? AND status = ?", link.EntryDomainID, model.StatusActive).First(&d).Error == nil {
			link.PublicURL = d.Scheme + "://" + d.Host + "/" + url.PathEscape(link.Code)
		}
	}
	return link, err
}

func (s *LinkService) Update(ctx context.Context, id uint64, input UpdateLinkInput) (model.Link, error) {
	link, err := s.Get(ctx, id)
	if err != nil {
		return model.Link{}, err
	}

	updates := map[string]interface{}{}
	if input.TargetURL != nil {
		if err := validateTargetURL(*input.TargetURL); err != nil {
			return model.Link{}, err
		}
		updates["target_url"] = *input.TargetURL
	}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.ExpireAt != nil {
		updates["expire_at"] = *input.ExpireAt
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.OnlineSchedule != nil {
		// 允许清空（传空字符串表示不限时段）
		if strings.TrimSpace(*input.OnlineSchedule) == "" {
			updates["online_schedule"] = nil
		} else {
			updates["online_schedule"] = *input.OnlineSchedule
		}
	}
	// 换入口域名
	if input.EntryDomainID != nil && *input.EntryDomainID != link.EntryDomainID {
		var domain model.Domain
		if err := s.db.WithContext(ctx).Where("id = ? AND type = ? AND status = ?", *input.EntryDomainID, model.DomainTypeEntry, model.StatusActive).First(&domain).Error; err != nil {
			return model.Link{}, ErrTargetUnavailable
		}
		updates["entry_domain_id"] = *input.EntryDomainID
	}
	// 换短码：旧码存入别名表，访问旧码自动跳转新码
	oldCode := link.Code
	if input.Code != nil && strings.TrimSpace(*input.Code) != "" && strings.TrimSpace(*input.Code) != link.Code {
		newCode := strings.TrimSpace(*input.Code)
		if !customCodePattern.MatchString(newCode) {
			return model.Link{}, ErrInvalidCode
		}
		// 检查新码是否已被占用
		var count int64
		s.db.WithContext(ctx).Model(&model.Link{}).Where("code = ? AND id != ? AND deleted_at IS NULL", newCode, link.ID).Count(&count)
		if count > 0 {
			return model.Link{}, ErrCodeConflict
		}
		updates["code"] = newCode
	}

	if len(updates) > 0 {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&link).Updates(updates).Error; err != nil {
				return err
			}
			// 换码成功后，把旧码写入别名表
			if newCode, ok := updates["code"].(string); ok && newCode != oldCode {
				alias := model.LinkCodeAlias{OldCode: oldCode, LinkID: link.ID}
				// 用 ON DUPLICATE KEY 忽略重复
				tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&alias)
			}
			return nil
		})
		if err != nil {
			return model.Link{}, err
		}
		// 清旧码缓存
		_ = s.cache.Delete(ctx, oldCode)
		if newCode, ok := updates["code"].(string); ok {
			_ = s.cache.Delete(ctx, newCode)
		}
	}

	return s.Get(ctx, id)
}

func (s *LinkService) Delete(ctx context.Context, id uint64) error {
	link, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&link).Error; err != nil {
		return err
	}
	return s.cache.Delete(ctx, link.Code)
}

// LegacyResolve 旧版 URL 兼容：按 legacy_id 或 code 查找链接并解析。
// 用于 /s/?key=xxx、/common/channel/redirect/?cid=xxx 等旧格式。
func (s *LinkService) LegacyResolve(ctx context.Context, identifier string, byID bool) (ResolveResult, error) {
	var link model.Link
	var err error
	if byID {
		id, parseErr := strconv.ParseUint(identifier, 10, 64)
		if parseErr != nil {
			return ResolveResult{}, ErrLinkNotFound
		}
		err = s.db.WithContext(ctx).Where("legacy_id = ? AND deleted_at IS NULL", id).First(&link).Error
	} else {
		err = s.db.WithContext(ctx).Where("code = ? AND deleted_at IS NULL", identifier).First(&link).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 查别名表
			var alias model.LinkCodeAlias
			aliasErr := s.db.WithContext(ctx).Where("old_code = ?", identifier).First(&alias).Error
			if aliasErr == nil {
				err = s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", alias.LinkID).First(&link).Error
			}
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ResolveResult{}, ErrLinkNotFound
	}
	if err != nil {
		return ResolveResult{}, err
	}
	return s.resolveFromModel(ctx, link)
}

func (s *LinkService) resolveFromModel(ctx context.Context, link model.Link) (ResolveResult, error) {
	if link.Status == model.LinkStatusDisabled {
		return ResolveResult{}, ErrLinkDisabled
	}
	if link.ExpireAt != nil && time.Now().After(*link.ExpireAt) {
		return ResolveResult{}, ErrLinkExpired
	}
	switch link.Type {
	case model.LinkTypeShort:
		if link.TargetURL == nil || *link.TargetURL == "" {
			return ResolveResult{}, ErrTargetUnavailable
		}
		return ResolveResult{Link: cache.CachedLink{ID: link.ID, Code: link.Code, Type: link.Type, Status: link.Status, TargetURL: *link.TargetURL}, TargetURL: *link.TargetURL, Status: http.StatusFound}, nil
	case model.LinkTypeChannel:
		if link.TargetURL == nil || *link.TargetURL == "" {
			return ResolveResult{}, ErrTargetUnavailable
		}
		channel, err := s.getChannelConfig(ctx, link.ID)
		if err != nil {
			return ResolveResult{}, err
		}
		cached := cache.CachedLink{ID: link.ID, Code: link.Code, Type: link.Type, Status: link.Status, TargetURL: *link.TargetURL}
		fillCachedUTM(&cached, channel)
		return ResolveResult{Link: cached, TargetURL: appendUTM(*link.TargetURL, cached), Status: http.StatusFound}, nil
	case model.LinkTypeLiveQR:
		landingURL, err := s.landingURL(ctx, link)
		if err != nil {
			return ResolveResult{}, err
		}
		return ResolveResult{Link: cache.CachedLink{ID: link.ID, Code: link.Code, Type: link.Type, Status: link.Status, TargetURL: landingURL}, TargetURL: landingURL, Status: http.StatusFound}, nil
	default:
		return ResolveResult{}, ErrUnsupportedLink
	}
}

func (s *LinkService) Resolve(ctx context.Context, code string) (ResolveResult, error) {
	cached, err := s.cache.Get(ctx, code)
	if err == nil {
		return resolveCached(cached)
	}
	if !errors.Is(err, cache.ErrCacheMiss) {
		return ResolveResult{}, err
	}

	var link model.Link
	dbErr := s.db.WithContext(ctx).Where("code = ? AND deleted_at IS NULL", code).First(&link).Error
	if errors.Is(dbErr, gorm.ErrRecordNotFound) {
		// 查别名表：旧码自动跳转到新码
		var alias model.LinkCodeAlias
		aliasErr := s.db.WithContext(ctx).Where("old_code = ?", code).First(&alias).Error
		if aliasErr != nil {
			return ResolveResult{}, ErrLinkNotFound
		}
		dbErr = s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", alias.LinkID).First(&link).Error
		if dbErr != nil {
			return ResolveResult{}, ErrLinkNotFound
		}
	}
	if dbErr != nil && !errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return ResolveResult{}, dbErr
	}

	cached = cache.CachedLink{
		ID:              link.ID,
		Code:            link.Code,
		Type:            link.Type,
		Status:          link.Status,
		AccessRule:      link.AccessRule,
		ExpireAt:        link.ExpireAt,
		LandingPageID:   link.LandingPageID,
		TransitDomainID: link.TransitDomainID,
		LandingDomainID: link.LandingDomainID,
	}
	if link.TargetURL != nil {
		cached.TargetURL = *link.TargetURL
	}
	if link.Type == model.LinkTypeChannel {
		channel, err := s.getChannelConfig(ctx, link.ID)
		if err != nil {
			return ResolveResult{}, err
		}
		fillCachedUTM(&cached, channel)
	}
	if link.Type == model.LinkTypeLiveQR {
		landingURL, err := s.landingURL(ctx, link)
		if err != nil {
			return ResolveResult{}, err
		}
		cached.TargetURL = landingURL
	}
	if err := s.cache.Set(ctx, cached); err != nil {
		return ResolveResult{}, err
	}

	return resolveCached(cached)
}

func resolveCached(link cache.CachedLink) (ResolveResult, error) {
	if link.Status == model.LinkStatusDisabled {
		return ResolveResult{}, ErrLinkDisabled
	}
	if link.ExpireAt != nil && time.Now().After(*link.ExpireAt) {
		return ResolveResult{}, ErrLinkExpired
	}
	if link.Type != model.LinkTypeShort {
		if link.Type == model.LinkTypeChannel {
			return ResolveResult{
				Link:      link,
				TargetURL: appendUTM(link.TargetURL, link),
				Status:    http.StatusFound,
			}, nil
		}
		if link.Type == model.LinkTypeLiveQR {
			if link.TargetURL == "" {
				return ResolveResult{}, ErrTargetUnavailable
			}
			return ResolveResult{Link: link, TargetURL: link.TargetURL, Status: http.StatusFound}, nil
		}
		return ResolveResult{}, ErrUnsupportedLink
	}
	if link.TargetURL == "" {
		return ResolveResult{}, ErrTargetUnavailable
	}

	return ResolveResult{
		Link:      link,
		TargetURL: link.TargetURL,
		Status:    http.StatusFound,
	}, nil
}

func (s *LinkService) landingURL(ctx context.Context, link model.Link) (string, error) {
	if link.LandingDomainID == nil {
		return "", ErrTargetUnavailable
	}
	var domain model.Domain
	err := s.db.WithContext(ctx).First(&domain, *link.LandingDomainID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrDomainNotFound
	}
	if err != nil {
		return "", err
	}
	return domain.Scheme + "://" + domain.Host + "/" + link.Code, nil
}

func (s *LinkService) createRoutingStrategy(tx *gorm.DB, linkID uint64, input *StrategyInput, linkType string) error {
	if linkType != model.LinkTypeLiveQR {
		return nil
	}
	// kami 提取页无需轮换目标
	if input == nil || len(input.Targets) == 0 {
		return nil
	}
	return s.routing.CreateStrategy(tx, linkID, input)
}

func (s *LinkService) getChannelConfig(ctx context.Context, linkID uint64) (model.ChannelConfig, error) {
	var channel model.ChannelConfig
	err := s.db.WithContext(ctx).Where("link_id = ?", linkID).First(&channel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ChannelConfig{}, nil
	}
	return channel, err
}

func validateTargetURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ErrInvalidTargetURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidTargetURL
	}
	if parsed.Host == "" {
		return ErrInvalidTargetURL
	}
	return nil
}

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func normalizeCreateError(err error) error {
	if isDuplicateError(err) {
		return ErrCodeConflict
	}
	return err
}

func createChannelConfig(tx *gorm.DB, linkID uint64, input *ChannelInput, linkType string) error {
	if linkType != model.LinkTypeChannel {
		return nil
	}
	config := model.ChannelConfig{LinkID: linkID}
	if input != nil {
		config.UTMSource = optionalString(input.UTMSource)
		config.UTMMedium = optionalString(input.UTMMedium)
		config.UTMCampaign = optionalString(input.UTMCampaign)
		config.UTMTerm = optionalString(input.UTMTerm)
		config.UTMContent = optionalString(input.UTMContent)
	}
	return tx.Create(&config).Error
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func fillCachedUTM(cached *cache.CachedLink, channel model.ChannelConfig) {
	cached.UTMSource = derefString(channel.UTMSource)
	cached.UTMMedium = derefString(channel.UTMMedium)
	cached.UTMCampaign = derefString(channel.UTMCampaign)
	cached.UTMTerm = derefString(channel.UTMTerm)
	cached.UTMContent = derefString(channel.UTMContent)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func appendUTM(rawURL string, link cache.CachedLink) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsed.Query()
	addQueryIfMissing(query, "utm_source", link.UTMSource)
	addQueryIfMissing(query, "utm_medium", link.UTMMedium)
	addQueryIfMissing(query, "utm_campaign", link.UTMCampaign)
	addQueryIfMissing(query, "utm_term", link.UTMTerm)
	addQueryIfMissing(query, "utm_content", link.UTMContent)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func addQueryIfMissing(query url.Values, key string, value string) {
	if value == "" || query.Has(key) {
		return
	}
	query.Set(key, value)
}
