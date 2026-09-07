package model

import "time"

type Material struct {
	ID        uint64 `gorm:"primaryKey"`
	Name      string `gorm:"size:255"`
	Path      string `gorm:"size:255"`
	CreatedAt time.Time
}

type ShareCard struct {
	ID          uint64 `gorm:"primaryKey"`
	DomainID    uint64
	Title       string `gorm:"size:128"`
	Description string `gorm:"size:500"`
	ImageURL    string `gorm:"type:text"`
	TargetURL   string `gorm:"type:text"`
	Status      string `gorm:"size:16"`
	Visits      uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublicURL   string `gorm:"-"`
}

type WechatAccount struct {
	ID     uint64 `gorm:"primaryKey"`
	AppID  string `gorm:"size:128"`
	Secret string `gorm:"type:text" json:"-"`
}
