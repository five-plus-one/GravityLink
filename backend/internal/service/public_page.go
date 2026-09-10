package service

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

type PublicPageService struct {
	db        *gorm.DB
	templates *template.Template
}

type PublicPageView struct {
	SiteName   string
	Title      string
	Message    string
	Footer     string
	ICP        string
	PoliceICP  string
}

func NewPublicPageService(db *gorm.DB, templates *template.Template) *PublicPageService {
	return &PublicPageService{db: db, templates: templates}
}

func (s *PublicPageService) Home(ctx context.Context) (string, int, string) {
	values := s.configs(ctx)
	// 自动跳转：配置了 public.home.redirect_url 时直接 302
	if redirectURL := values["public.home.redirect_url"]; redirectURL != "" {
		return "", http.StatusFound, redirectURL
	}
	view := s.view(ctx, "home", PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接服务正在运行",
		Message:  "这是短链接访问入口，请使用完整短链接访问目标内容。",
		Footer:   "GravityLink",
	})
	return s.render(view, http.StatusOK), http.StatusOK, ""
}

func (s *PublicPageService) NotFound(ctx context.Context) (string, int) {
	view := s.view(ctx, "not_found", PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接不存在或已失效",
		Message:  "请检查链接是否完整，或联系链接提供方确认当前链接状态。",
		Footer:   "GravityLink",
	})
	return s.render(view, http.StatusNotFound), http.StatusNotFound
}

func (s *PublicPageService) Gone(ctx context.Context) (string, int) {
	view := s.view(ctx, "gone", PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接已过期",
		Message:  "该链接已超过有效期，无法继续访问。",
		Footer:   "GravityLink",
	})
	return s.render(view, http.StatusGone), http.StatusGone
}

// AccessDenied 返回访问受限提示页（UA 访问限制不满足时）。
// 403 状态码；Message 告知使用何种方式打开，例如「请用微信客户端打开此链接」。
func (s *PublicPageService) AccessDenied(ctx context.Context, message string) (string, int) {
	view := s.view(ctx, "access_denied", PublicPageView{
		SiteName: "GravityLink",
		Title:    "访问受限",
		Message:  message,
		Footer:   "GravityLink",
	})
	return s.render(view, http.StatusForbidden), http.StatusForbidden
}

func (s *PublicPageService) view(ctx context.Context, page string, fallback PublicPageView) PublicPageView {
	values := s.configs(ctx)
	return PublicPageView{
		SiteName:  firstPublicValue(values["public.site_name"], fallback.SiteName),
		Title:     firstPublicValue(values["public."+page+".title"], fallback.Title),
		Message:   firstPublicValue(values["public."+page+".message"], fallback.Message),
		Footer:    firstPublicValue(values["public.footer"], fallback.Footer),
		ICP:       values["public.icp"],
		PoliceICP: values["public.police_icp"],
	}
}

func (s *PublicPageService) configs(ctx context.Context) map[string]string {
	var items []model.SystemConfig
	if err := s.db.WithContext(ctx).Where("key_name LIKE ?", "public.%").Find(&items).Error; err != nil {
		return map[string]string{}
	}
	values := make(map[string]string, len(items))
	for _, item := range items {
		values[item.KeyName] = item.Value
	}
	return values
}

func firstPublicValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *PublicPageService) render(view PublicPageView, status int) string {
	if s.templates == nil {
		return "templates not loaded"
	}
	statusText := "HTTP " + http.StatusText(status)
	if status == http.StatusOK {
		statusText = ""
	}
	data := map[string]interface{}{
		"Title":      view.Title,
		"Message":    view.Message,
		"Footer":     view.Footer,
		"ICP":        view.ICP,
		"PoliceICP":  view.PoliceICP,
		"StatusText": statusText,
	}
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, "public.html", data); err != nil {
		return errors.New("render public page failed: " + err.Error()).Error()
	}
	return buf.String()
}
