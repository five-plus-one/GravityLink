package router

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"strings"
)

const hiddenMaterialsKey = "materials.hidden"

func hiddenMaterials(db *gorm.DB) ([]uint64, error) {
	var row model.SystemConfig
	ids := []uint64{}
	err := db.Where("key_name = ?", hiddenMaterialsKey).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return ids, nil
	}
	if err != nil {
		return ids, err
	}
	err = json.Unmarshal([]byte(row.Value), &ids)
	return ids, err
}
func registerMaterialManagement(admin *gin.RouterGroup, db *gorm.DB) {
	admin.PUT("/materials/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		var input struct {
			Name string `json:"name"`
		}
		if c.ShouldBindJSON(&input) != nil || strings.TrimSpace(input.Name) == "" || len([]rune(input.Name)) > 255 {
			response.Error(c, 400, 4001, "名称不能为空，最多255字")
			return
		}
		r := db.WithContext(c.Request.Context()).Model(&model.Material{}).Where("id = ?", id).Update("name", strings.TrimSpace(input.Name))
		if r.Error != nil || r.RowsAffected == 0 {
			response.Error(c, 404, 4004, "素材不存在")
			return
		}
		response.OK(c, gin.H{"updated": true})
	})
	admin.POST("/materials/remove", func(c *gin.Context) {
		var input struct {
			IDs []uint64 `json:"ids"`
		}
		if c.ShouldBindJSON(&input) != nil || len(input.IDs) == 0 || len(input.IDs) > 5000 {
			response.Error(c, 400, 4001, "请选择要移出的素材")
			return
		}
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			row := model.SystemConfig{KeyName: hiddenMaterialsKey, Value: "[]"}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key_name = ?", hiddenMaterialsKey).First(&row).Error; err != nil {
				return err
			}
			ids, err := hiddenMaterials(tx)
			if err != nil {
				return err
			}
			seen := map[uint64]bool{}
			for _, id := range ids {
				seen[id] = true
			}
			for _, id := range input.IDs {
				if !seen[id] {
					ids = append(ids, id)
					seen[id] = true
				}
			}
			b, _ := json.Marshal(ids)
			return tx.Model(&model.SystemConfig{}).Where("key_name = ?", hiddenMaterialsKey).Update("value", string(b)).Error
		})
		if err != nil {
			response.Error(c, 400, 4001, "移出失败，请重试")
			return
		}
		response.OK(c, gin.H{"removed": true})
	})
}
