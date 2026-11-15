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
	// Isolate the pending batch from concurrent LPUSH producers. A failed write
	// stays in this durable processing list and is retried after restart.
	processingKey := service.AccessStreamKey + ":processing"
	if err := c.redis.Eval(ctx, `
if redis.call('EXISTS', KEYS[2]) == 0 and redis.call('EXISTS', KEYS[1]) == 1 then
 return redis.call('RENAME', KEYS[1], KEYS[2])
end
return 0`, []string{service.AccessStreamKey, processingKey}).Err(); err != nil {
		return err
	}
	payloads, err := c.redis.LRange(ctx, processingKey, 0, 99).Result()
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
	return c.redis.LTrim(ctx, processingKey, int64(len(payloads)), -1).Err()
}

func accessLogFromEvent(event service.AccessEvent) model.AccessLog {
	referer := boundedLogString(event.Referer, 2048)
	info := service.ParseUserAgent(event.UserAgent)
	return model.AccessLog{
		LinkID:     event.LinkID,
		VisitedAt:  event.VisitedAt,
		IP:         event.IP,
		Country:    boundedLogString(event.Country, 64),
		Province:   boundedLogString(event.Province, 64),
		City:       boundedLogString(event.City, 64),
		ISP:        boundedLogString(event.ISP, 64),
		Device:     info.Device,
		OS:         boundedLogString(info.OS, 64),
		Browser:    boundedLogString(info.Browser, 64),
		Referer:    referer,
		SourceApp:  boundedLogString(event.SourceApp, 64),
		ViaTransit: event.ViaTransit,
	}
}

func optionalLogString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func boundedLogString(value string, limit int) *string {
	chars := []rune(value)
	if len(chars) > limit {
		value = string(chars[:limit])
	}
	return optionalLogString(value)
}
