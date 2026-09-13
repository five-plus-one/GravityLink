package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gravitylink/backend/internal/model"
)

const labelCatalogKey = "links.label-catalog"

type LabelCatalog struct {
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
}
type LabelMutation struct {
	Kind   string   `json:"kind"`
	Action string   `json:"action"`
	Names  []string `json:"names"`
	Target string   `json:"target"`
}
type BulkLinksInput struct {
	IDs      []uint64 `json:"ids"`
	Category *string  `json:"category"`
	Tags     []string `json:"tags"`
	TagMode  string   `json:"tag_mode"`
	Status   *string  `json:"status"`
}

func uniqueLabels(values []string) []string {
	set := map[string]bool{}
	result := []string{}
	for _, v := range values {
		if v != "" && !set[v] {
			set[v] = true
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
func readCatalog(tx *gorm.DB) (LabelCatalog, error) {
	var row model.SystemConfig
	catalog := LabelCatalog{Categories: []string{}, Tags: []string{}}
	err := tx.Where("key_name = ?", labelCatalogKey).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return catalog, err
	}
	if err == nil {
		if err = json.Unmarshal([]byte(row.Value), &catalog); err != nil {
			return catalog, err
		}
	}
	var links []model.Link
	if err = tx.Select("id", "category", "tags").Find(&links).Error; err != nil {
		return catalog, err
	}
	for _, link := range links {
		catalog.Categories = append(catalog.Categories, link.Category)
		catalog.Tags = append(catalog.Tags, link.Tags...)
	}
	catalog.Categories = uniqueLabels(catalog.Categories)
	catalog.Tags = uniqueLabels(catalog.Tags)
	return catalog, nil
}
func (s *LinkService) Labels(ctx context.Context) (LabelCatalog, error) {
	return readCatalog(s.db.WithContext(ctx))
}
func (s *LinkService) MutateLabels(ctx context.Context, input LabelMutation) error {
	if input.Kind != "category" && input.Kind != "tag" {
		return errors.New("请选择分类或标签")
	}
	if input.Action != "add" && input.Action != "remove" && input.Action != "move" {
		return errors.New("操作无效")
	}
	input.Target = strings.TrimSpace(input.Target)
	limit := 80
	if input.Kind == "tag" {
		limit = 40
	}
	if len(input.Names) == 0 || len(input.Names) > 500 {
		return errors.New("请选择要管理的项目")
	}
	selected := map[string]bool{}
	for _, v := range input.Names {
		v = strings.TrimSpace(v)
		if v == "" || len([]rune(v)) > limit {
			return errors.New("名称为空或过长")
		}
		selected[v] = true
	}
	if len([]rune(input.Target)) > limit || (input.Action == "move" && input.Target == "") {
		return errors.New("请填写目标名称")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A shared row serializes directory changes and bulk association changes.
		if err := lockLabels(tx); err != nil {
			return err
		}
		catalog, err := readCatalog(tx)
		if err != nil {
			return err
		}
		values := catalog.Categories
		if input.Kind == "tag" {
			values = catalog.Tags
		}
		next := []string{}
		for _, v := range values {
			if input.Action == "add" || !selected[v] {
				next = append(next, v)
			}
		}
		if input.Action == "add" {
			for v := range selected {
				next = append(next, v)
			}
		}
		if input.Action == "move" {
			next = append(next, input.Target)
		}
		if input.Action != "add" {
			var links []model.Link
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&links).Error; err != nil {
				return err
			}
			for _, link := range links {
				updates := map[string]any{}
				if input.Kind == "category" && selected[link.Category] {
					target := ""
					if input.Action == "move" {
						target = input.Target
					}
					updates["category"] = target
				}
				if input.Kind == "tag" {
					tags := []string{}
					changed := false
					for _, tag := range link.Tags {
						if selected[tag] {
							changed = true
						} else {
							tags = append(tags, tag)
						}
					}
					if changed {
						if input.Action == "move" {
							tags = append(tags, input.Target)
						}
						_, tags, err = normalizeLinkLabels("", uniqueLabels(tags))
						if err != nil {
							return err
						}
						b, _ := json.Marshal(tags)
						updates["tags"] = string(b)
					}
				}
				if len(updates) > 0 {
					if err := tx.Model(&model.Link{}).Where("id = ?", link.ID).Updates(updates).Error; err != nil {
						return err
					}
				}
			}
		}
		if input.Kind == "tag" {
			catalog.Tags = uniqueLabels(next)
		} else {
			catalog.Categories = uniqueLabels(next)
		}
		b, _ := json.Marshal(catalog)
		return tx.Model(&model.SystemConfig{}).Where("key_name = ?", labelCatalogKey).Update("value", string(b)).Error
	})
}
func lockLabels(tx *gorm.DB) error {
	row := model.SystemConfig{KeyName: labelCatalogKey, Value: `{"categories":[],"tags":[]}`}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return err
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key_name = ?", labelCatalogKey).First(&model.SystemConfig{}).Error
}
func (s *LinkService) BulkUpdate(ctx context.Context, input BulkLinksInput) error {
	if len(input.IDs) == 0 || len(input.IDs) > 5000 {
		return errors.New("请选择1至5000条链接")
	}
	if input.Status != nil && *input.Status != "active" && *input.Status != "disabled" {
		return errors.New("状态无效")
	}
	if input.TagMode != "" && input.TagMode != "append" && input.TagMode != "remove" && input.TagMode != "replace" {
		return errors.New("标签操作无效")
	}
	category := ""
	if input.Category != nil {
		category = *input.Category
	}
	category, tags, err := normalizeLinkLabels(category, input.Tags)
	if err != nil {
		return err
	}
	ids := map[uint64]bool{}
	for _, id := range input.IDs {
		ids[id] = true
	}
	var changed []model.Link
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLabels(tx); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", input.IDs).Find(&changed).Error; err != nil {
			return err
		}
		if len(changed) != len(ids) {
			return ErrLinkNotFound
		}
		for _, link := range changed {
			updates := map[string]any{}
			if input.Category != nil {
				updates["category"] = category
			}
			if input.Status != nil {
				updates["status"] = *input.Status
			}
			if input.TagMode != "" {
				next := tags
				if input.TagMode == "append" {
					next = uniqueLabels(append(link.Tags, tags...))
				}
				if input.TagMode == "remove" {
					next = []string{}
					for _, v := range link.Tags {
						found := false
						for _, t := range tags {
							if v == t {
								found = true
							}
						}
						if !found {
							next = append(next, v)
						}
					}
				}
				_, next, err = normalizeLinkLabels("", next)
				if err != nil {
					return err
				}
				b, _ := json.Marshal(next)
				updates["tags"] = string(b)
			}
			if len(updates) > 0 {
				if err := tx.Model(&model.Link{}).Where("id = ?", link.ID).Updates(updates).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err == nil && input.Status != nil && s.cache != nil {
		for _, link := range changed {
			_ = s.cache.Delete(ctx, link.Code)
		}
	}
	return err
}
