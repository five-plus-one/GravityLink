package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type LogArchiver struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewLogArchiver(db *gorm.DB, logger *slog.Logger) *LogArchiver {
	return &LogArchiver{db: db, logger: logger}
}

func (a *LogArchiver) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if now.Hour() != 2 {
				continue
			}
			if err := a.Archive(ctx, now.AddDate(0, 0, -90)); err != nil {
				a.logger.Warn("archive access logs failed", "error", err)
			}
		}
	}
}

func (a *LogArchiver) Archive(ctx context.Context, before time.Time) error {
	cutoff := before.Format("2006-01-02 15:04:05")
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("INSERT INTO access_logs_archive SELECT * FROM access_logs WHERE visited_at < ?", cutoff).Error; err != nil {
			return err
		}
		return tx.Exec("DELETE FROM access_logs WHERE visited_at < ?", cutoff).Error
	})
}
