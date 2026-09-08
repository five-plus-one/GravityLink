package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"strconv"
	"strings"
	"time"
)

type targetBatchInput struct {
	URLs      []string `json:"urls"`
	ScanLimit *uint    `json:"scan_limit"`
}

func validateTargetBatch(in targetBatchInput) error {
	if len(in.URLs) == 0 || len(in.URLs) > 100 {
		return fmt.Errorf("每批需包含 1–100 条图片地址")
	}
	if in.ScanLimit != nil && *in.ScanLimit == 0 {
		return fmt.Errorf("阈值必须大于 0，留空为不限")
	}
	seen := map[string]bool{}
	for i, raw := range in.URLs {
		u := strings.TrimSpace(raw)
		if !validTargetURL(u) || len(u) > 2048 {
			return fmt.Errorf("第 %d 条地址无效", i+1)
		}
		if seen[u] {
			return fmt.Errorf("第 %d 条地址重复", i+1)
		}
		seen[u] = true
	}
	return nil
}

func validTargetURL(raw string) bool {
	if httpURL(raw) {
		return true
	}
	return strings.HasPrefix(raw, "/uploads/") && !strings.Contains(raw, "..") && !strings.ContainsAny(raw, "?#\\")
}

func registerTargetRoutes(admin *gin.RouterGroup, deps Dependencies) {
	admin.POST("/links/:id/targets/:target/reset-count", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		targetID, err := strconv.ParseUint(c.Param("target"), 10, 64)
		if err != nil || targetID == 0 {
			response.Error(c, 400, 4001, "二维码 ID 无效")
			return
		}
		var input struct {
			ExpectedCount *uint  `json:"expected_count"`
			Confirmation  string `json:"confirmation"`
		}
		if c.ShouldBindJSON(&input) != nil || input.ExpectedCount == nil || input.Confirmation != "RESET TARGET COUNT" {
			response.Error(c, 400, 4001, "请确认当前计数")
			return
		}
		var target model.RoutingTarget
		if deps.DB.Joins("JOIN routing_strategies ON routing_strategies.id = routing_targets.strategy_id").Joins("JOIN links ON links.id = routing_strategies.link_id AND links.deleted_at IS NULL").Where("routing_targets.id = ? AND routing_strategies.link_id = ?", targetID, id).First(&target).Error != nil {
			response.Error(c, 404, 4004, "二维码不存在")
			return
		}
		conflict := errors.New("count changed")
		err = deps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&model.RoutingTarget{}).Where("id = ? AND status = ? AND scan_count = ?", target.ID, "disabled", *input.ExpectedCount).UpdateColumn("scan_count", 0)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return conflict
			}
			userValue, _ := c.Get(middleware.ContextUserKey)
			user := userValue.(middleware.AuthUser)
			detail, _ := json.Marshal(map[string]any{"old_count": *input.ExpectedCount, "new_count": 0, "link_id": id})
			typ, tid := "routing_target", fmt.Sprint(target.ID)
			return tx.Create(&model.AuditLog{UserID: &user.ID, Action: "target.reset_count", TargetType: &typ, TargetID: &tid, Detail: detail}).Error
		})
		if errors.Is(err, conflict) {
			response.Error(c, 409, 4009, "请先停用二维码并刷新当前计数")
			return
		}
		if err != nil {
			response.Error(c, 500, 5000, "重置失败，计数未变更")
			return
		}
		response.OK(c, gin.H{"reset": true})
	})
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
	// P1：删除二维码目标（带审计日志）。批量导入贴错不再只能停用。
	admin.DELETE("/links/:id/targets/:target", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		targetID, err := strconv.ParseUint(c.Param("target"), 10, 64)
		if err != nil || targetID == 0 {
			response.Error(c, 400, 4001, "二维码 ID 无效")
			return
		}
		var s model.RoutingStrategy
		if deps.DB.Where("link_id = ?", id).First(&s).Error != nil {
			response.Error(c, 404, 4004, "活码不存在")
			return
		}
		var target model.RoutingTarget
		if deps.DB.Where("id = ? AND strategy_id = ?", targetID, s.ID).First(&target).Error != nil {
			response.Error(c, 404, 4004, "二维码不存在")
			return
		}
		err = deps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&target).Error; err != nil {
				return err
			}
			userValue, _ := c.Get(middleware.ContextUserKey)
			user := userValue.(middleware.AuthUser)
			detail, _ := json.Marshal(map[string]any{"link_id": id, "target_url": target.TargetURL, "label": target.Label})
			typ, tid := "routing_target", fmt.Sprint(target.ID)
			return tx.Create(&model.AuditLog{UserID: &user.ID, Action: "target.delete", TargetType: &typ, TargetID: &tid, Detail: detail}).Error
		})
		if err != nil {
			response.Error(c, 500, 5000, "删除失败")
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
	// P1：批量设置扫码阈值（仅作用于启用的目标）。
	admin.PUT("/links/:id/targets/batch-limit", func(c *gin.Context) {
		s, ok := strategy(c)
		if !ok {
			return
		}
		var in struct {
			ScanLimit *uint `json:"scan_limit"`
		}
		if c.ShouldBindJSON(&in) != nil || (in.ScanLimit != nil && *in.ScanLimit == 0) {
			response.Error(c, 400, 4001, "阈值必须大于 0，留空为不限")
			return
		}
		result := deps.DB.Model(&model.RoutingTarget{}).Where("strategy_id = ?", s.ID).Update("scan_limit", in.ScanLimit)
		if result.Error != nil {
			response.Error(c, 500, 5000, "批量设置失败")
			return
		}
		response.OK(c, gin.H{"updated": result.RowsAffected})
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
		if c.ShouldBindJSON(&in) != nil || !validTargetURL(in.TargetURL) || len(in.Label) > 128 || len(in.Owner) > 128 || in.Weight < 1 || (in.Status != "active" && in.Status != "disabled") {
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
	admin.POST("/links/:id/targets/batch", func(c *gin.Context) {
		s, ok := strategy(c)
		if !ok {
			return
		}
		var in targetBatchInput
		if c.ShouldBindJSON(&in) != nil {
			response.Error(c, 400, 4001, "批量格式无效")
			return
		}
		if err := validateTargetBatch(in); err != nil {
			response.Error(c, 400, 4001, err.Error())
			return
		}
		items := make([]model.RoutingTarget, 0, len(in.URLs))
		err := deps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			for i, raw := range in.URLs {
				label := fmt.Sprintf("批量二维码 %d", i+1)
				items = append(items, model.RoutingTarget{StrategyID: s.ID, Label: &label, TargetURL: strings.TrimSpace(raw), Weight: 1, ScanLimit: in.ScanLimit, Status: "active"})
			}
			return tx.Create(&items).Error
		})
		if err != nil {
			response.Error(c, 500, 5000, "批量保存失败，未添加任何二维码")
			return
		}
		response.OK(c, gin.H{"items": items, "total": len(items)})
	})
}
