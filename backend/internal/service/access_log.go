package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const AccessStreamKey = "access:stream"

type AccessRecorder struct {
	redis *redis.Client
}

type AccessEvent struct {
	LinkID     uint64    `json:"link_id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"ua"`
	Referer    string    `json:"referer"`
	VisitedAt  time.Time `json:"visited_at"`
	ViaTransit bool      `json:"via_transit"`
}

func NewAccessRecorder(redis *redis.Client) *AccessRecorder {
	return &AccessRecorder{redis: redis}
}

func (r *AccessRecorder) RecordAsync(event AccessEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		day := event.VisitedAt.Format("20060102")
		hour := event.VisitedAt.Format("15")
		_ = r.redis.Incr(ctx, fmt.Sprintf("stat:pv:%d:%s", event.LinkID, day)).Err()
		_ = r.redis.PFAdd(ctx, fmt.Sprintf("stat:uv:%d:%s", event.LinkID, day), event.IP).Err()
		_ = r.redis.Incr(ctx, fmt.Sprintf("stat:hourly:%d:%s:%s", event.LinkID, day, hour)).Err()

		payload, err := json.Marshal(event)
		if err != nil {
			return
		}
		_ = r.redis.LPush(ctx, AccessStreamKey, payload).Err()
	}()
}
