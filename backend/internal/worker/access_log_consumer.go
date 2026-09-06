package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/service"
)

type AccessLogConsumer struct {
	db     *gorm.DB
	redis  *redis.Client
	logger *slog.Logger
}

func NewAccessLogConsumer(db *gorm.DB, redis *redis.Client, logger *slog.Logger) *AccessLogConsumer {
	return &AccessLogConsumer{db: db, redis: redis, logger: logger}
}

func (c *AccessLogConsumer) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.consumeBatch(ctx); err != nil {
				c.logger.Warn("consume access logs failed", "error", err)
			}
		}
	}
}

func (c *AccessLogConsumer) consumeBatch(ctx context.Context) error {
	payloads, err := c.redis.LRange(ctx, service.AccessStreamKey, 0, 99).Result()
	if err != nil {
		return err
	}
	if len(payloads) == 0 {
		return nil
	}

	logs := make([]model.AccessLog, 0, len(payloads))
	for _, payload := range payloads {
		var event service.AccessEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			c.logger.Warn("skip invalid access event", "error", err)
			continue
		}
		logs = append(logs, accessLogFromEvent(event))
	}

	if len(logs) > 0 {
		if err := c.db.WithContext(ctx).CreateInBatches(logs, 100).Error; err != nil {
			return err
		}
	}
	return c.redis.LTrim(ctx, service.AccessStreamKey, int64(len(payloads)), -1).Err()
}

func accessLogFromEvent(event service.AccessEvent) model.AccessLog {
	referer := optionalLogString(event.Referer)
	info := service.ParseUserAgent(event.UserAgent)
	return model.AccessLog{
		LinkID:     event.LinkID,
		VisitedAt:  event.VisitedAt,
		IP:         event.IP,
		Country:    optionalLogString(event.Country),
		Province:   optionalLogString(event.Province),
		City:       optionalLogString(event.City),
		ISP:        optionalLogString(event.ISP),
		Device:     info.Device,
		OS:         optionalLogString(info.OS),
		Browser:    optionalLogString(info.Browser),
		Referer:    referer,
		ViaTransit: event.ViaTransit,
	}
}

func optionalLogString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
