package router

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerDomainRoutes(group *gin.RouterGroup, domains *service.DomainService) {
	group.GET("/domains", func(c *gin.Context) {
		items, err := domains.List(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 5000, "list domains failed")
			return
		}
		response.OK(c, gin.H{"items": items, "total": len(items)})
	})

	group.POST("/domains", func(c *gin.Context) {
		var input service.DomainInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4101, "invalid request body")
			return
		}

		domain, err := domains.Create(c.Request.Context(), input)
		writeDomainResult(c, domain, err)
	})

	group.PUT("/domains/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		var input service.DomainInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, 4101, "invalid request body")
			return
		}

		domain, err := domains.Update(c.Request.Context(), id, input)
		writeDomainResult(c, domain, err)
	})

	group.DELETE("/domains/:id", func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := domains.Delete(c.Request.Context(), id); err != nil {
			writeDomainError(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
}

func writeDomainResult(c *gin.Context, data interface{}, err error) {
	if err != nil {
		writeDomainError(c, err)
		return
	}
	response.OK(c, data)
}

func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDomainNotFound):
		response.Error(c, http.StatusNotFound, 4104, "domain not found")
	case errors.Is(err, service.ErrCodeConflict):
		response.Error(c, http.StatusConflict, 4101, "domain already exists")
	case errors.Is(err, service.ErrInvalidDomainHost):
		response.Error(c, http.StatusBadRequest, 4102, "invalid domain host")
	case errors.Is(err, service.ErrInvalidDomainType):
		response.Error(c, http.StatusBadRequest, 4103, "invalid domain type")
	case errors.Is(err, service.ErrDomainInUse):
		response.Error(c, http.StatusConflict, 4105, "domain in use")
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "internal server error")
	}
}
