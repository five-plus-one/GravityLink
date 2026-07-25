package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
)

func registerUserRoutes(group *gin.RouterGroup, db *gorm.DB) {
	group.GET("/users", func(c *gin.Context) {
		var users []model.User
		if err := db.Order("id ASC").Find(&users).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list users failed")
			return
		}
		items := make([]map[string]any, 0, len(users))
		for _, user := range users {
			items = append(items, userPayload(user))
		}
		response.OK(c, map[string]any{"items": items, "total": len(items)})
	})

	super := group.Group("/users")
	super.Use(middleware.RequireSuperAdmin())
	super.PUT("/:id/role", func(c *gin.Context) {
		var input struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil ||
			(input.Role != model.UserRoleAdmin && input.Role != model.UserRoleUser) {
			response.Error(c, http.StatusBadRequest, 4000, "role must be admin or user")
			return
		}
		if err := db.Model(&model.User{}).Where("id = ? AND role <> ?", c.Param("id"), model.UserRoleSuperAdmin).
			Update("role", input.Role).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "update role failed")
			return
		}
		response.OK(c, map[string]bool{"updated": true})
	})
	super.PUT("/:id/status", func(c *gin.Context) {
		var input struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil ||
			(input.Status != model.StatusActive && input.Status != model.StatusDisabled) {
			response.Error(c, http.StatusBadRequest, 4000, "status must be active or disabled")
			return
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.User{}).Where("id = ? AND role <> ?", c.Param("id"), model.UserRoleSuperAdmin).
				Update("status", input.Status).Error; err != nil {
				return err
			}
			if input.Status == model.StatusDisabled {
				return tx.Where("user_id = ?", c.Param("id")).Delete(&model.AuthSession{}).Error
			}
			return nil
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "update status failed")
			return
		}
		response.OK(c, map[string]bool{"updated": true})
	})
}

func userPayload(user model.User) map[string]any {
	email := ""
	subject := ""
	if user.Email != nil {
		email = *user.Email
	}
	if user.SSOID != nil {
		subject = *user.SSOID
	}
	return map[string]any{
		"id": user.ID, "auth_source": user.AuthSource, "subject": subject, "username": user.Username,
		"email": email, "role": user.Role, "status": user.Status, "last_login_at": user.LastLoginAt,
		"created_at": user.CreatedAt,
	}
}
