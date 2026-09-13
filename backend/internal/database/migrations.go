package database

import (
	"fmt"
	"gorm.io/gorm"
	"gravitylink/backend/internal/model"
	"time"
)

type schemaVersion struct {
	Version   uint64    `gorm:"primaryKey;autoIncrement:false"`
	AppliedAt time.Time `gorm:"not null"`
}

// MigrateVersions keeps additive upgrade steps replayable after partial DDL failure.
// The connection-scoped lock serializes simultaneous application starts.
func MigrateVersions(db *gorm.DB) error {
	return db.Connection(func(tx *gorm.DB) error {
		tx = tx.Session(&gorm.Session{NewDB: true})
		var locked int
		if err := tx.Raw("SELECT GET_LOCK('gravitylink_schema_upgrade',60)").Scan(&locked).Error; err != nil {
			return err
		}
		if locked != 1 {
			return fmt.Errorf("schema migration lock unavailable")
		}
		defer tx.Exec("SELECT RELEASE_LOCK('gravitylink_schema_upgrade')")
		if err := tx.AutoMigrate(&schemaVersion{}); err != nil {
			return err
		}
		const version uint64 = 2026091401
		var count int64
		if err := tx.Model(&schemaVersion{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return err
		}

		for _, field := range []string{"Category", "Tags"} {
			if !tx.Migrator().HasColumn(&model.Link{}, field) {
				if err := tx.Migrator().AddColumn(&model.Link{}, field); err != nil {
					return err
				}
			}
		}
		if count > 0 {
			return nil
		}
		return tx.Create(&schemaVersion{Version: version, AppliedAt: time.Now()}).Error
	})
}
