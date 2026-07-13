package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const LinkCacheTTL = 24 * time.Hour

var ErrCacheMiss = errors.New("link cache miss")

type LinkCache struct {
	redis *redis.Client
}

type CachedLink struct {
	ID              uint64
	Code            string
	Type            string
	TargetURL       string
	Status          string
	ExpireAt        *time.Time
	LandingPageID   *uint64
	TransitDomainID *uint64
	LandingDomainID *uint64
	UTMSource       string
	UTMMedium       string
	UTMCampaign     string
	UTMTerm         string
	UTMContent      string
}

func NewLinkCache(redis *redis.Client) *LinkCache {
	return &LinkCache{redis: redis}
}

func (c *LinkCache) Get(ctx context.Context, code string) (CachedLink, error) {
	values, err := c.redis.HGetAll(ctx, linkCacheKey(code)).Result()
	if err != nil {
		return CachedLink{}, err
	}
	if len(values) == 0 {
		return CachedLink{}, ErrCacheMiss
	}

	link := CachedLink{
		Code:        values["code"],
		Type:        values["type"],
		TargetURL:   values["target_url"],
		Status:      values["status"],
		UTMSource:   values["utm_source"],
		UTMMedium:   values["utm_medium"],
		UTMCampaign: values["utm_campaign"],
		UTMTerm:     values["utm_term"],
		UTMContent:  values["utm_content"],
	}

	id, err := strconv.ParseUint(values["id"], 10, 64)
	if err != nil {
		return CachedLink{}, err
	}
	link.ID = id

	if values["expire_at_unix"] != "" {
		expireAt, err := strconv.ParseInt(values["expire_at_unix"], 10, 64)
		if err != nil {
			return CachedLink{}, err
		}
		t := time.Unix(expireAt, 0)
		link.ExpireAt = &t
	}

	link.LandingPageID = parseOptionalUint(values["landing_page_id"])
	link.TransitDomainID = parseOptionalUint(values["transit_domain_id"])
	link.LandingDomainID = parseOptionalUint(values["landing_domain_id"])
	return link, nil
}

func (c *LinkCache) Set(ctx context.Context, link CachedLink) error {
	values := map[string]interface{}{
		"id":                strconv.FormatUint(link.ID, 10),
		"code":              link.Code,
		"type":              link.Type,
		"target_url":        link.TargetURL,
		"status":            link.Status,
		"expire_at_unix":    optionalTimeUnix(link.ExpireAt),
		"landing_page_id":   optionalUint(link.LandingPageID),
		"transit_domain_id": optionalUint(link.TransitDomainID),
		"landing_domain_id": optionalUint(link.LandingDomainID),
		"utm_source":        link.UTMSource,
		"utm_medium":        link.UTMMedium,
		"utm_campaign":      link.UTMCampaign,
		"utm_term":          link.UTMTerm,
		"utm_content":       link.UTMContent,
	}

	key := linkCacheKey(link.Code)
	if err := c.redis.HSet(ctx, key, values).Err(); err != nil {
		return err
	}
	return c.redis.Expire(ctx, key, LinkCacheTTL).Err()
}

func (c *LinkCache) Delete(ctx context.Context, code string) error {
	return c.redis.Del(ctx, linkCacheKey(code)).Err()
}

func linkCacheKey(code string) string {
	return "link:cache:" + strings.ToLower(code)
}

func optionalTimeUnix(value *time.Time) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(value.Unix(), 10)
}

func optionalUint(value *uint64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatUint(*value, 10)
}

func parseOptionalUint(value string) *uint64 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}
