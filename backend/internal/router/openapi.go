package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

// registerAPIKeyRoutes 管理端 API Key CRUD（/api/admin/api-keys）。
func registerAPIKeyRoutes(group *gin.RouterGroup, svc *service.APIKeyService) {
	group.GET("/api-keys", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		keys, err := svc.ListKeys(c.Request.Context(), user.ID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list api keys failed")
			return
		}
		response.OK(c, gin.H{"items": keys, "total": len(keys)})
	})

	group.POST("/api-keys", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		var input service.CreateKeyInput
		if c.ShouldBindJSON(&input) != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid input")
			return
		}
		result, err := svc.CreateKey(c.Request.Context(), input, user.ID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, 4000, err.Error())
			return
		}
		// Token/HmacSecret 仅本次返回；后续 List/Get 只见 hash，不可恢复
		response.OK(c, gin.H{
			"id":           result.ApiKey.ID,
			"name":         result.ApiKey.Name,
			"token":        result.Token,
			"hmac_secret":  result.HmacSecret,
			"sign_enabled": result.SignEnabled,
		})
	})

	group.PUT("/api-keys/:id", func(c *gin.Context) {
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		var input service.UpdateKeyInput
		if c.ShouldBindJSON(&input) != nil {
			response.Error(c, http.StatusBadRequest, 4001, "invalid input")
			return
		}
		key, err := svc.UpdateKey(c.Request.Context(), id, input)
		if err != nil {
			writeAPIKeyError(c, err)
			return
		}
		response.OK(c, key)
	})

	group.DELETE("/api-keys/:id", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		if err := svc.DeleteKey(c.Request.Context(), id, user.ID); err != nil {
			writeAPIKeyError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
}

// registerOpenAPIRoutes 公开 API（/api/v1/open）：Bearer token 鉴权，无需 session cookie。
func registerOpenAPIRoutes(group *gin.RouterGroup, svc *service.APIKeyService, links *service.LinkService, db *gorm.DB) {
	group.POST("/short-links", func(c *gin.Context) {
		bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if bearer == "" {
			response.Error(c, http.StatusUnauthorized, 4401, "missing Authorization header")
			return
		}
		key, err := svc.ValidateKey(
			c.Request.Context(), bearer,
			c.ClientIP(),
			c.GetHeader("X-Timestamp"),
			readBodyForSign(c),
			c.GetHeader("X-Signature"),
		)
		if err != nil {
			writeAPIKeyError(c, err)
			return
		}

		var input struct {
			TargetURL     string `json:"target_url"`
			Code          string `json:"code"`
			Title         string `json:"title"`
			EntryDomainID uint64 `json:"entry_domain_id"`
		}
		if c.ShouldBindJSON(&input) != nil || input.TargetURL == "" {
			response.Error(c, http.StatusUnprocessableEntity, 4220, "target_url is required")
			return
		}

		// 未指定入口域名时取域名库中第一条 active 入口域名
		entryDomainID := input.EntryDomainID
		if entryDomainID == 0 {
			entryDomainID = firstEntryDomainID(c.Request.Context(), db)
			if entryDomainID == 0 {
				response.Error(c, http.StatusUnprocessableEntity, 4220, "no entry domain configured")
				return
			}
		}

		link, err := links.Create(c.Request.Context(), service.CreateLinkInput{
			TargetURL:     input.TargetURL,
			Code:          input.Code,
			Title:         input.Title,
			EntryDomainID: entryDomainID,
			CreatedBy:     key.CreatedBy,
		})
		if err != nil {
			response.Error(c, http.StatusUnprocessableEntity, 4220, mapAPIError(err))
			return
		}
		svc.IncrKeyUsage(c.Request.Context(), key.ID)
		response.OK(c, gin.H{
			"id":   link.ID,
			"code": link.Code,
			"url":  fmt.Sprintf("%s/%s", linkEntryDomain(c.Request.Context(), db, entryDomainID), link.Code),
		})
	})
}

// ---- helpers ----

func getAuthUser(c *gin.Context) (middleware.AuthUser, bool) {
	v, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		response.Error(c, http.StatusUnauthorized, 4401, "unauthorized")
		return middleware.AuthUser{}, false
	}
	user, ok := v.(middleware.AuthUser)
	if !ok {
		response.Error(c, http.StatusInternalServerError, 5000, "auth context invalid")
		return middleware.AuthUser{}, false
	}
	return user, true
}

func parseIDParam(c *gin.Context) (uint64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, 4001, "invalid id")
		return 0, false
	}
	return id, true
}

func readBodyForSign(c *gin.Context) string {
	if c.GetHeader("X-Signature") == "" {
		return ""
	}
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	return string(body)
}

func firstEntryDomainID(ctx context.Context, db *gorm.DB) uint64 {
	var d model.Domain
	if err := db.WithContext(ctx).Where("type = ? AND status = ?", model.DomainTypeEntry, model.StatusActive).Order("id ASC").First(&d).Error; err != nil {
		return 0
	}
	return d.ID
}

func linkEntryDomain(ctx context.Context, db *gorm.DB, domainID uint64) string {
	var d model.Domain
	if err := db.WithContext(ctx).First(&d, domainID).Error; err != nil {
		return ""
	}
	return d.Scheme + "://" + d.Host
}

func mapAPIError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "duplicate") {
		return "code already exists"
	}
	return msg
}

func writeAPIKeyError(c *gin.Context, err error) {
	switch err {
	case service.ErrAPIKeyNotFound:
		response.Error(c, http.StatusNotFound, 4004, "api key not found")
	case service.ErrAPIKeyDisabled:
		response.Error(c, http.StatusForbidden, 4403, "api key disabled")
	case service.ErrAPIKeyExpired:
		response.Error(c, http.StatusForbidden, 4403, "api key expired")
	case service.ErrAPIKeyIPDenied:
		response.Error(c, http.StatusForbidden, 4403, "ip not in api key whitelist")
	case service.ErrAPIKeyQuotaExceeded:
		response.Error(c, http.StatusConflict, 4409, "api key quota exceeded")
	case service.ErrAPIKeySignInvalid:
		response.Error(c, http.StatusUnauthorized, 4401, "invalid request signature")
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "internal error")
	}
}
