package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

// ApplyQRInspect 将识别结果写入素材行。
func ApplyQRInspect(db *gorm.DB, item *model.Material, result QRInspectResult) error {
	now := time.Now()
	return db.Model(item).Updates(map[string]any{
		"qr_content":          result.Content,
		"qr_kind":             result.Kind,
		"suggested_expire_at": result.SuggestedExpireAt,
		"expire_source":       result.ExpireSource,
		"inspected_at":        &now,
	}).Error
}

// MaterialByPath 按路径或文件名查素材。
func MaterialByPath(ctx context.Context, db *gorm.DB, path string) (model.Material, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return model.Material{}, false, nil
	}
	var item model.Material
	err := db.WithContext(ctx).Where("path = ?", path).First(&item).Error
	if err == nil {
		return item, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Material{}, false, err
	}
	base := path
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		base = path[i+1:]
	}
	if base == "" {
		return model.Material{}, false, nil
	}
	err = db.WithContext(ctx).Where("path LIKE ?", "%/"+base).Or("path LIKE ?", base).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Material{}, false, nil
	}
	if err != nil {
		return model.Material{}, false, err
	}
	return item, true, nil
}

// SuggestExpireForTargetURL 目标 URL 对应素材的建议到期时间；未识别时尝试本地懒识别。
func SuggestExpireForTargetURL(ctx context.Context, db *gorm.DB, uploadsDir, targetURL string, defaultDays int) (*time.Time, string) {
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" || !IsLocalUploadURL(targetURL) {
		return nil, ExpireSourceNone
	}
	item, ok, err := MaterialByPath(ctx, db, targetURL)
	if err != nil || !ok {
		return nil, ExpireSourceNone
	}
	if item.InspectedAt == nil && uploadsDir != "" {
		if data, err := readLocalUpload(uploadsDir, item.Path); err == nil && len(data) > 0 {
			res := InspectQR(data, defaultDays)
			if err := ApplyQRInspect(db, &item, res); err == nil {
				return res.SuggestedExpireAt, res.ExpireSource
			}
		}
	}
	return item.SuggestedExpireAt, item.ExpireSource
}

func readLocalUpload(uploadsDir, path string) ([]byte, error) {
	name := path
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		name = path[i+1:]
	}
	name = filepath.Base(name)
	if name == "." || name == ".." || name == "" {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(filepath.Join(uploadsDir, name))
}

// InspectBytesAndSave 对上传字节做识别并回写素材。
func InspectBytesAndSave(db *gorm.DB, materialID uint64, data []byte, defaultDays int) (QRInspectResult, error) {
	res := InspectQR(data, defaultDays)
	var item model.Material
	if err := db.First(&item, materialID).Error; err != nil {
		return res, err
	}
	if err := ApplyQRInspect(db, &item, res); err != nil {
		return res, err
	}
	return res, nil
}

// DefaultExpireDaysFromConfig 读取默认群码有效天数。
func DefaultExpireDaysFromConfig(ctx context.Context, n *Notifier) int {
	if n == nil {
		return 7
	}
	cfg, err := n.LoadEvents(ctx)
	if err != nil || cfg.QRDefaultExpireDays <= 0 {
		return 7
	}
	return cfg.QRDefaultExpireDays
}

// IsLocalUploadURL 是否站内素材地址。
func IsLocalUploadURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "/uploads/") {
		return !strings.Contains(raw, "..")
	}
	return strings.Contains(raw, "/uploads/") && !strings.Contains(raw, "://")
}
