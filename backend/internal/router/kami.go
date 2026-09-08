package router

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

// registerKamiRoutes 注册卡密管理路由（/api/admin/kami/*）。
func registerKamiRoutes(group *gin.RouterGroup, svc *service.KamiService) {
	// ---- 项目 CRUD ----
	group.GET("/kami/projects", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		projects, err := svc.ListProjects(c.Request.Context(), user.ID)
		if err != nil {
			response.Error(c, 500, 5000, "list projects failed")
			return
		}
		// 附加实时统计（剩余/已发）
		type projectWithStats struct {
			ProjectID uint64 `json:"project_id"`
			Remaining int64  `json:"remaining"`
			Issued    int64  `json:"issued"`
		}
		type projectDTO struct {
			ID        uint64  `json:"id"`
			Title     string  `json:"title"`
			Type      string  `json:"type"`
			Status    string  `json:"status"`
			Remaining int64   `json:"remaining"`
			Issued    int64   `json:"issued"`
		}
		dtos := make([]projectDTO, len(projects))
		for i, p := range projects {
			rem, iss, _ := svc.ProjectStats(c.Request.Context(), p.ID)
			dtos[i] = projectDTO{ID: p.ID, Title: p.Title, Type: p.Type, Status: p.Status, Remaining: rem, Issued: iss}
		}
		response.OK(c, gin.H{"items": dtos, "total": len(dtos)})
	})

	group.POST("/kami/projects", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		var input service.CreateKamiProjectInput
		if c.ShouldBindJSON(&input) != nil {
			response.Error(c, 400, 4001, "invalid input")
			return
		}
		p, err := svc.CreateProject(c.Request.Context(), input, user.ID)
		if err != nil {
			response.Error(c, 400, 4000, err.Error())
			return
		}
		response.OK(c, p)
	})

	group.PUT("/kami/projects/:id", func(c *gin.Context) {
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		var input struct {
			Title        *string `json:"title"`
			Password     *string `json:"password"`
			RepeatPolicy *string `json:"repeat_policy"`
			RepeatIntervalSec *uint `json:"repeat_interval_sec"`
			Status       *string `json:"status"`
		}
		if c.ShouldBindJSON(&input) != nil {
			response.Error(c, 400, 4001, "invalid input")
			return
		}
		p, err := svc.UpdateProject(c.Request.Context(), id, input)
		if err != nil {
			writeKamiError(c, err)
			return
		}
		response.OK(c, p)
	})

	group.DELETE("/kami/projects/:id", func(c *gin.Context) {
		user, ok := getAuthUser(c)
		if !ok {
			return
		}
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		if err := svc.DeleteProject(c.Request.Context(), id, user.ID); err != nil {
			writeKamiError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})

	// ---- 卡密列表 / 导入 ----
	group.GET("/kami/projects/:id/items", func(c *gin.Context) {
		projectID, ok := parseID(c)
		if !ok {
			return
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		items, total, err := svc.ListItems(c.Request.Context(), projectID, limit, offset)
		if err != nil {
			response.Error(c, 500, 5000, "list items failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": total})
	})

	group.POST("/kami/projects/:id/items/import", func(c *gin.Context) {
		projectID, ok := parseID(c)
		if !ok {
			return
		}
		var input struct {
			Items []service.ImportKamiItem `json:"items"`
		}
		if c.ShouldBindJSON(&input) != nil || len(input.Items) == 0 {
			response.Error(c, 400, 4001, "items list required")
			return
		}
		result, err := svc.ImportItems(c.Request.Context(), projectID, input.Items)
		if err != nil {
			response.Error(c, 400, 4000, err.Error())
			return
		}
		response.OK(c, result)
	})

	group.DELETE("/kami/items/:id", func(c *gin.Context) {
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		if err := svc.DeleteItem(c.Request.Context(), id); err != nil {
			writeKamiError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})

	// ---- 提取记录 ----
	group.GET("/kami/projects/:id/issuances", func(c *gin.Context) {
		projectID, ok := parseIDParam(c)
		if !ok {
			return
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		items, total, err := svc.ListIssuances(c.Request.Context(), projectID, limit, offset)
		if err != nil {
			response.Error(c, 500, 5000, "list issuances failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": total})
	})

	group.DELETE("/kami/issuances/:id", func(c *gin.Context) {
		id, ok := parseIDParam(c)
		if !ok {
			return
		}
		if err := svc.DeleteIssuance(c.Request.Context(), id); err != nil {
			response.Error(c, 500, 5000, "delete failed")
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
}

// registerKamiPublicRoutes 卡密公开提取接口（/api/v1/kami/:id/issue）。
func registerKamiPublicRoutes(group *gin.RouterGroup, svc *service.KamiService) {
	group.POST("/kami/:id/issue", func(c *gin.Context) {
		projectID, ok := parseID(c)
		if !ok {
			return
		}
		var input struct {
			Password string `json:"password"`
		}
		c.ShouldBindJSON(&input) // password 可选，由 service 层根据项目配置判断

		result, err := svc.Issue(
			c.Request.Context(), projectID, input.Password,
			c.ClientIP(), c.Request.UserAgent(), "",
		)
		if err != nil {
			writeKamiError(c, err)
			return
		}
		response.OK(c, result)
	})
}

func writeKamiError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case err == service.ErrKamiProjectNotFound || err == service.ErrKamiItemNotFound:
		response.Error(c, 404, 4004, msg)
	case err == service.ErrKamiEmpty:
		response.Error(c, 410, 4101, "已领完")
	case err == service.ErrKamiPasswordRequired:
		response.Error(c, 400, 4001, "请输入提取口令")
	case err == service.ErrKamiPasswordWrong:
		response.Error(c, 403, 4403, "提取口令错误")
	case err == service.ErrKamiRepeatNotAllowed:
		response.Error(c, 403, 4403, "该 IP 已提取过，请勿重复操作")
	case err == service.ErrKamiTooFrequent:
		response.Error(c, 429, 4290, "提取过于频繁，请稍后再试")
	default:
		response.Error(c, 500, 5000, msg)
	}
}
