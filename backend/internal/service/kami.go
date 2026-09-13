package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gravitylink/backend/internal/model"
)

var (
	ErrKamiProjectNotFound  = errors.New("kami project not found")
	ErrKamiItemNotFound     = errors.New("kami item not found")
	ErrKamiEmpty            = errors.New("no items available")
	ErrKamiPasswordRequired = errors.New("password required")
	ErrKamiPasswordWrong    = errors.New("password incorrect")
	ErrKamiRepeatNotAllowed = errors.New("repeat extraction not allowed")
	ErrKamiTooFrequent      = errors.New("extraction too frequent")
	ErrKamiIssued           = errors.New("already issued") // 乐观锁竞态兜底
)

type KamiService struct {
	db *gorm.DB
}

func NewKamiService(db *gorm.DB) *KamiService {
	return &KamiService{db: db}
}

// ---- 项目 CRUD ----

type CreateKamiProjectInput struct {
	Title             string  `json:"title"`
	Type              string  `json:"type"`
	Password          *string `json:"password"`      // 提取口令，可选
	RepeatPolicy      string  `json:"repeat_policy"` // never / allow
	RepeatIntervalSec uint    `json:"repeat_interval_sec"`
}

func (s *KamiService) CreateProject(ctx context.Context, input CreateKamiProjectInput, createdBy uint64) (*model.KamiProject, error) {
	p := &model.KamiProject{
		Title:             input.Title,
		Type:              input.Type,
		Password:          input.Password,
		RepeatPolicy:      input.RepeatPolicy,
		RepeatIntervalSec: input.RepeatIntervalSec,
		Status:            model.StatusActive,
		CreatedBy:         createdBy,
	}
	if p.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if p.RepeatPolicy != "allow" {
		p.RepeatPolicy = "never"
	}
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *KamiService) ListProjects(ctx context.Context, createdBy uint64) ([]model.KamiProject, error) {
	var projects []model.KamiProject
	return projects, s.db.WithContext(ctx).Where("created_by = ?", createdBy).Order("id DESC").Find(&projects).Error
}

func (s *KamiService) UpdateProject(ctx context.Context, id uint64, input struct {
	Title             *string `json:"title"`
	Password          *string `json:"password"`
	RepeatPolicy      *string `json:"repeat_policy"`
	RepeatIntervalSec *uint   `json:"repeat_interval_sec"`
	Status            *string `json:"status"`
}) (*model.KamiProject, error) {
	var p model.KamiProject
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, ErrKamiProjectNotFound
	}
	updates := map[string]any{}
	if input.Title != nil {
		if strings.TrimSpace(*input.Title) == "" || len([]rune(*input.Title)) > 128 {
			return nil, fmt.Errorf("项目标题不能为空，最多128字")
		}
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if input.Password != nil {
		if len([]rune(*input.Password)) > 128 {
			return nil, fmt.Errorf("口令最多128字")
		}
		updates["password"] = input.Password
	}
	if input.RepeatPolicy != nil && (*input.RepeatPolicy == "never" || *input.RepeatPolicy == "allow") {
		updates["repeat_policy"] = *input.RepeatPolicy
	}
	if input.RepeatIntervalSec != nil {
		updates["repeat_interval_sec"] = *input.RepeatIntervalSec
	}
	if input.Status != nil && (*input.Status == "active" || *input.Status == "disabled") {
		updates["status"] = *input.Status
	}
	if err := s.db.WithContext(ctx).Model(&p).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &p, s.db.WithContext(ctx).First(&p, id).Error
}

func (s *KamiService) DeleteProject(ctx context.Context, id, createdBy uint64) error {
	result := s.db.WithContext(ctx).Where("id = ? AND created_by = ?", id, createdBy).Delete(&model.KamiProject{})
	if result.RowsAffected == 0 {
		return ErrKamiProjectNotFound
	}
	return result.Error
}

// ---- 卡密批量导入 ----

type ImportKamiItem struct {
	Content     string `json:"content"`
	Note        string `json:"note"`
	ExpiresText string `json:"expires_text"`
}

type ImportResult struct {
	Imported int `json:"imported"`
	Dup      int `json:"dup"`
	Failed   int `json:"failed"`
}

func (s *KamiService) ImportItems(ctx context.Context, projectID uint64, items []ImportKamiItem) (ImportResult, error) {
	if len(items) == 0 {
		return ImportResult{}, fmt.Errorf("empty list")
	}
	if len(items) > 1000 {
		return ImportResult{}, fmt.Errorf("max 1000 items per import")
	}
	var result ImportResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project model.KamiProject
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, projectID).Error; err != nil {
			return ErrKamiProjectNotFound
		}
		for _, input := range items {
			content := trim(input.Content)
			if content == "" {
				result.Failed++
				continue
			}
			var count int64
			if err := tx.Model(&model.KamiItem{}).Where("project_id = ? AND content = ?", projectID, content).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				result.Dup++
				continue
			}
			item := model.KamiItem{ProjectID: projectID, Content: content, Status: "unissued"}
			if input.Note != "" {
				item.Note = &input.Note
			}
			if input.ExpiresText != "" {
				item.ExpiresText = &input.ExpiresText
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			result.Imported++
		}
		return nil
	})
	return result, err
}

func (s *KamiService) ListItems(ctx context.Context, projectID uint64, limit, offset int) ([]model.KamiItem, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items := []model.KamiItem{}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.KamiItem{}).Where("project_id = ?", projectID)
	q.Count(&total)
	err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (s *KamiService) DeleteItem(ctx context.Context, id uint64) error {
	result := s.db.WithContext(ctx).Delete(&model.KamiItem{}, id)
	if result.RowsAffected == 0 {
		return ErrKamiItemNotFound
	}
	return result.Error
}

// ---- 提取逻辑（公开 API） ----

type IssueResult struct {
	Content string `json:"content"`
}

// Issue 发放一条卡密（原子：并发安全，乐观锁防重复发放）。
// visitorIP/UA/Device 用于提取记录（风控审计）。
func (s *KamiService) Issue(ctx context.Context, projectID uint64, password, visitorIP, visitorUA, visitorDevice string) (*IssueResult, error) {
	var p model.KamiProject
	if err := s.db.WithContext(ctx).First(&p, projectID).Error; err != nil {
		return nil, ErrKamiProjectNotFound
	}
	if p.Status != model.StatusActive {
		return nil, ErrKamiProjectNotFound
	}

	// 原子发码：一条事务内取 + 置 issued，用 FOR UPDATE SKIP LOCKED 防竞态
	var item model.KamiItem
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the project before checking the visitor's previous issuance.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, projectID).Error; err != nil {
			return err
		}
		if p.Status != model.StatusActive {
			return ErrKamiProjectNotFound
		} // 口令校验
		if p.Password != nil && *p.Password != "" {
			if password == "" {
				return ErrKamiPasswordRequired
			}
			if password != *p.Password {
				return ErrKamiPasswordWrong
			}
		}

		// 重复提取校验（同 IP+同项目）
		if p.RepeatPolicy != "allow" {
			var count int64
			if err := tx.Model(&model.KamiIssuance{}).
				Where("project_id = ? AND visitor_ip = ?", projectID, visitorIP).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrKamiRepeatNotAllowed
			}
		} else if p.RepeatIntervalSec > 0 {
			var last model.KamiIssuance
			err := tx.
				Where("project_id = ? AND visitor_ip = ?", projectID, visitorIP).
				Order("id DESC").First(&last).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil && time.Since(last.CreatedAt) < time.Duration(p.RepeatIntervalSec)*time.Second {
				return ErrKamiTooFrequent
			}
		}

		// 最旧未发放的卡密优先（FIFO）
		result := tx.Raw(
			"SELECT * FROM kami_items WHERE project_id = ? AND status = 'unissued' ORDER BY id ASC LIMIT 1 FOR UPDATE SKIP LOCKED",
			projectID,
		).Scan(&item)
		if result.RowsAffected == 0 {
			return ErrKamiEmpty
		}
		if result.Error != nil {
			return result.Error
		}
		issuance := model.KamiIssuance{
			ProjectID:     projectID,
			KamiItemID:    item.ID,
			VisitorIP:     &visitorIP,
			VisitorUA:     &visitorUA,
			VisitorDevice: &visitorDevice,
		}
		if err := tx.Create(&issuance).Error; err != nil {
			return err
		}
		return tx.Model(&model.KamiItem{}).Where("id = ?", item.ID).UpdateColumns(map[string]any{
			"status":    "issued",
			"issued_at": time.Now(),
			"issued_ip": visitorIP,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &IssueResult{Content: item.Content}, nil
}

// ---- 提取记录 ----

func (s *KamiService) ListIssuances(ctx context.Context, projectID uint64, limit, offset int) ([]model.KamiIssuance, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items := []model.KamiIssuance{}
	var total int64
	q := s.db.WithContext(ctx).Model(&model.KamiIssuance{}).Where("project_id = ?", projectID)
	q.Count(&total)
	err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (s *KamiService) DeleteIssuance(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&model.KamiIssuance{}, id).Error
}

// ProjectStats 返回剩余/已发/总数。
func (s *KamiService) ProjectStats(ctx context.Context, projectID uint64) (remaining, issued int64, err error) {
	s.db.WithContext(ctx).Model(&model.KamiItem{}).Where("project_id = ? AND status = 'unissued'", projectID).Count(&remaining)
	s.db.WithContext(ctx).Model(&model.KamiItem{}).Where("project_id = ? AND status = 'issued'", projectID).Count(&issued)
	return
}

func trim(s string) string {
	// 内联 strings.TrimSpace 以避免额外 import（本文件多处调用）
	start, end := 0, len(s)
	for start < end && s[start] <= ' ' {
		start++
	}
	for end > start && s[end-1] <= ' ' {
		end--
	}
	return s[start:end]
}
