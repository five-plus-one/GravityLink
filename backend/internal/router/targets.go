package router

import (
	"github.com/gin-gonic/gin"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"time"
)

func registerTargetRoutes(admin *gin.RouterGroup, deps Dependencies) {
	strategy := func(c *gin.Context) (model.RoutingStrategy, bool) {
		id, ok := parseID(c)
		if !ok {
			return model.RoutingStrategy{}, false
		}
		var link model.Link
		var s model.RoutingStrategy
		if deps.DB.Where("id = ? AND type = ?", id, model.LinkTypeLiveQR).First(&link).Error != nil || deps.DB.Where("link_id = ?", id).First(&s).Error != nil {
			response.Error(c, 404, 4004, "活码不存在")
			return s, false
		}
		return s, true
	}
	admin.GET("/links/:id/targets", func(c *gin.Context) {
		s, ok := strategy(c)
		if !ok {
			return
		}
		var items []model.RoutingTarget
		if deps.DB.Where("strategy_id = ?", s.ID).Order("priority DESC, id ASC").Find(&items).Error != nil {
			response.Error(c, 500, 5000, "读取失败")
			return
		}
		response.OK(c, gin.H{"mode": s.Mode, "items": items})
	})
	admin.PUT("/links/:id/strategy", func(c *gin.Context) {
		s, ok := strategy(c)
		if !ok {
			return
		}
		var in struct {
			Mode string `json:"mode"`
		}
		if c.ShouldBindJSON(&in) != nil || (in.Mode != "round_robin" && in.Mode != "weighted") {
			response.Error(c, 400, 4001, "策略无效")
			return
		}
		if deps.DB.Model(&s).Update("mode", in.Mode).Error != nil {
			response.Error(c, 500, 5000, "保存失败")
			return
		}
		response.OK(c, gin.H{"saved": true})
	})
	save := func(c *gin.Context) {
		s, ok := strategy(c)
		if !ok {
			return
		}
		var in struct {
			ID        uint64
			Label     string
			TargetURL string
			Weight    uint
			ScanLimit *uint
			Priority  int
			Status    string
			ExpireAt  *time.Time
			Owner     string
		}
		if c.ShouldBindJSON(&in) != nil || !httpURL(in.TargetURL) || len(in.Label) > 128 || len(in.Owner) > 128 || in.Weight < 1 || (in.Status != "active" && in.Status != "disabled") {
			response.Error(c, 400, 4001, "目标配置无效")
			return
		}
		var t model.RoutingTarget
		if in.ID != 0 {
			if deps.DB.Where("id = ? AND strategy_id = ?", in.ID, s.ID).First(&t).Error != nil {
				response.Error(c, 404, 4004, "目标不存在")
				return
			}
		}
		t.StrategyID = s.ID
		t.Label = &in.Label
		t.TargetURL = in.TargetURL
		t.Weight = in.Weight
		t.ScanLimit = in.ScanLimit
		t.Priority = in.Priority
		t.Status = in.Status
		t.ExpireAt = in.ExpireAt
		t.Owner = in.Owner
		var saveErr error
		if t.ID == 0 {
			saveErr = deps.DB.Create(&t).Error
		} else {
			saveErr = deps.DB.Model(&t).Updates(map[string]any{"label": t.Label, "target_url": t.TargetURL, "weight": t.Weight, "scan_limit": t.ScanLimit, "priority": t.Priority, "status": t.Status, "expire_at": t.ExpireAt, "owner": t.Owner}).Error
		}
		if saveErr != nil {
			response.Error(c, 500, 5000, "保存失败")
			return
		}
		response.OK(c, t)
	}
	admin.POST("/links/:id/targets", save)
}
