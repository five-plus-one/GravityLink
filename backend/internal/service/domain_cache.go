package service

import (
	"errors"
	"net"
	"strings"
	"sync"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var ErrDomainNotFound = errors.New("domain not found")

type DomainCache struct {
	db      *gorm.DB
	mu      sync.RWMutex
	domains map[string]model.Domain
}

func NewDomainCache(db *gorm.DB) *DomainCache {
	return &DomainCache{
		db:      db,
		domains: make(map[string]model.Domain),
	}
}

func (c *DomainCache) Load() error {
	var domains []model.Domain
	if err := c.db.Where("status = ?", model.StatusActive).Find(&domains).Error; err != nil {
		return err
	}

	next := make(map[string]model.Domain, len(domains))
	for _, domain := range domains {
		next[strings.ToLower(strings.TrimSpace(domain.Host))] = domain
	}

	c.mu.Lock()
	c.domains = next
	c.mu.Unlock()
	return nil
}

func (c *DomainCache) Get(host string) (model.Domain, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domain, ok := c.domains[strings.ToLower(strings.TrimSpace(host))]
	if !ok {
		domain, ok = c.domains[normalizeHost(host)]
	}
	return domain, ok
}

func (c *DomainCache) Refresh() error {
	return c.Load()
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}

	return strings.TrimSuffix(host, ".")
}
