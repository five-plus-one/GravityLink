package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	UserRoleSuperAdmin = "super_admin"
	UserRoleAdmin      = "admin"
	UserRoleUser       = "user"

	AuthSourceLogto = "logto"
	AuthSourceLocal = "local"

	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusPending  = "pending"

	DomainTypeEntry   = "entry"
	DomainTypeTransit = "transit"
	DomainTypeLanding = "landing"

	LinkTypeShort   = "short"
	LinkTypeChannel = "channel"
	LinkTypeLiveQR  = "liveqr"

	LinkStatusActive   = "active"
	LinkStatusDisabled = "disabled"
	LinkStatusExpired  = "expired"
)

type User struct {
	ID           uint64  `gorm:"primaryKey;autoIncrement"`
	AuthSource   string  `gorm:"type:enum('logto','local');not null;default:logto"`
	SSOID        *string `gorm:"column:sso_id;size:128;uniqueIndex:uk_sso_id"`
	Username     string  `gorm:"size:64;not null;uniqueIndex:uk_username"`
	Email        *string `gorm:"size:128;uniqueIndex:uk_email"`
	PasswordHash *string `gorm:"size:255"`
	Role         string  `gorm:"type:enum('super_admin','admin','user');not null;default:user"`
	Status       string  `gorm:"type:enum('pending','active','disabled');not null;default:pending"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type AuthSession struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	UserID     uint64    `gorm:"not null;index:idx_session_user"`
	TokenHash  string    `gorm:"size:64;not null;uniqueIndex:uk_token_hash"`
	ExpiresAt  time.Time `gorm:"not null;index:idx_session_expire"`
	LastSeenAt *time.Time
	CreatedAt  time.Time `gorm:"not null"`
}

type InstallationState struct {
	ID            uint8 `gorm:"primaryKey"`
	Installed     bool  `gorm:"not null;default:false"`
	OwnerUserID   *uint64
	SchemaVersion uint `gorm:"not null;default:1"`
	InstalledAt   *time.Time
	UpdatedAt     time.Time `gorm:"not null"`
}

type AuditLog struct {
	ID         uint64          `gorm:"primaryKey;autoIncrement"`
	UserID     *uint64         `gorm:"index:idx_audit_user"`
	Action     string          `gorm:"size:64;not null;index:idx_audit_action"`
	TargetType *string         `gorm:"size:64"`
	TargetID   *string         `gorm:"size:128"`
	Detail     json.RawMessage `gorm:"type:json"`
	IP         *string         `gorm:"size:45"`
	CreatedAt  time.Time       `gorm:"not null;index:idx_audit_created"`
}

type Domain struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	Host      string         `gorm:"size:253;not null;uniqueIndex:uk_host"`
	Type      string         `gorm:"type:enum('entry','transit','landing');not null"`
	Scheme    string         `gorm:"type:enum('http','https');not null;default:https"`
	Remark    *string        `gorm:"size:255"`
	Status    string         `gorm:"type:enum('active','disabled');not null;default:active"`
	CreatedBy uint64         `gorm:"not null"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Link struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	Code            string `gorm:"size:32;not null;uniqueIndex:uk_code"`
	Type            string `gorm:"type:enum('short','channel','liveqr');not null;index:idx_type"`
	EntryDomainID   uint64 `gorm:"not null;index:idx_entry_domain"`
	TransitDomainID *uint64
	LandingDomainID *uint64
	TargetURL       *string `gorm:"type:text"`
	LandingPageID   *uint64
	Title           *string `gorm:"size:255"`
	ExpireAt        *time.Time
	Status          string         `gorm:"type:enum('active','disabled','expired');not null;default:active"`
	CreatedBy       uint64         `gorm:"not null;index:idx_created_by"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

type ChannelConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID      uint64    `gorm:"not null;uniqueIndex:uk_link_id"`
	UTMSource   *string   `gorm:"column:utm_source;size:128"`
	UTMMedium   *string   `gorm:"column:utm_medium;size:128"`
	UTMCampaign *string   `gorm:"column:utm_campaign;size:128"`
	UTMTerm     *string   `gorm:"column:utm_term;size:128"`
	UTMContent  *string   `gorm:"column:utm_content;size:128"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

type AccessLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID     uint64    `gorm:"not null;index:idx_link_visited"`
	VisitedAt  time.Time `gorm:"not null;index:idx_link_visited;index:idx_visited_at"`
	IP         string    `gorm:"size:45;not null"`
	Country    *string   `gorm:"size:64"`
	Province   *string   `gorm:"size:64"`
	City       *string   `gorm:"size:64"`
	ISP        *string   `gorm:"size:64"`
	Device     string    `gorm:"type:enum('mobile','tablet','desktop','bot','unknown');not null;default:unknown"`
	OS         *string   `gorm:"size:64"`
	Browser    *string   `gorm:"size:64"`
	Referer    *string   `gorm:"size:2048"`
	ViaTransit bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time `gorm:"not null"`
}

type LandingPage struct {
	ID        uint64          `gorm:"primaryKey;autoIncrement"`
	Template  string          `gorm:"type:enum('liveqr','redirect_notice','custom');not null"`
	Title     string          `gorm:"size:255;not null"`
	Content   json.RawMessage `gorm:"type:json;not null"`
	DomainID  uint64          `gorm:"not null;index:idx_domain_id"`
	CreatedBy uint64          `gorm:"not null"`
	CreatedAt time.Time       `gorm:"not null"`
	UpdatedAt time.Time       `gorm:"not null"`
	DeletedAt gorm.DeletedAt  `gorm:"index"`
}

type RoutingStrategy struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID    uint64    `gorm:"not null;uniqueIndex:uk_link_id"`
	Mode      string    `gorm:"type:enum('round_robin','weighted','time_based','geo_device');not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type RoutingTarget struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	StrategyID   uint64    `gorm:"not null;index:idx_strategy_id"`
	Label        *string   `gorm:"size:128"`
	TargetURL    string    `gorm:"type:text;not null"`
	Weight       uint      `gorm:"not null;default:1"`
	ScanLimit    *uint     `gorm:"column:scan_limit"`
	ScanCount    uint      `gorm:"not null;default:0"`
	TimeStart    *string   `gorm:"type:time"`
	TimeEnd      *string   `gorm:"type:time"`
	WeekdayMask  *uint8    `gorm:"column:weekday_mask"`
	GeoFilter    *string   `gorm:"type:json"`
	DeviceFilter *string   `gorm:"type:json"`
	Priority     int       `gorm:"not null;default:0"`
	Status       string    `gorm:"type:enum('active','disabled','exhausted');not null;default:active;index:idx_status"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

type StatDaily struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID   uint64    `gorm:"not null;uniqueIndex:uk_link_date"`
	StatDate time.Time `gorm:"type:date;not null;uniqueIndex:uk_link_date"`
	PV       uint64    `gorm:"column:pv;not null;default:0"`
	UV       uint64    `gorm:"column:uv;not null;default:0"`
	IPCount  uint64    `gorm:"column:ip_count;not null;default:0"`
}

type StatHourly struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID   uint64    `gorm:"not null;uniqueIndex:uk_link_date_hour"`
	StatDate time.Time `gorm:"type:date;not null;uniqueIndex:uk_link_date_hour"`
	StatHour uint8     `gorm:"not null;uniqueIndex:uk_link_date_hour"`
	PV       uint64    `gorm:"column:pv;not null;default:0"`
}

type StatGeo struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID   uint64    `gorm:"not null;uniqueIndex:uk_link_date_geo"`
	StatDate time.Time `gorm:"type:date;not null;uniqueIndex:uk_link_date_geo"`
	Country  string    `gorm:"size:64;not null;default:'';uniqueIndex:uk_link_date_geo"`
	Province string    `gorm:"size:64;not null;default:'';uniqueIndex:uk_link_date_geo"`
	PV       uint64    `gorm:"column:pv;not null;default:0"`
}

type StatDevice struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement"`
	LinkID   uint64    `gorm:"not null;uniqueIndex:uk_link_date_device"`
	StatDate time.Time `gorm:"type:date;not null;uniqueIndex:uk_link_date_device"`
	Device   string    `gorm:"size:32;not null;default:'';uniqueIndex:uk_link_date_device"`
	OS       string    `gorm:"size:64;not null;default:'';uniqueIndex:uk_link_date_device"`
	Browser  string    `gorm:"size:64;not null;default:'';uniqueIndex:uk_link_date_device"`
	PV       uint64    `gorm:"column:pv;not null;default:0"`
}

type SystemConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	KeyName     string    `gorm:"size:128;not null;uniqueIndex:uk_key_name"`
	Value       string    `gorm:"type:text;not null"`
	Description *string   `gorm:"size:255"`
	UpdatedAt   time.Time `gorm:"not null"`
}
