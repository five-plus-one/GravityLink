package service

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

type PublicPageService struct {
	db        *gorm.DB
	templates *template.Template
}

type PublicPageView struct {
	SiteName  string
	Title     string
	Message   string
	Footer    string
	ICP       string
	PoliceICP string
}

func NewPublicPageService(db *gorm.DB, templates *template.Template) *PublicPageService {
	return &PublicPageService{db: db, templates: templates}
}

func (s *PublicPageService) Home(ctx context.Context) (string, int) {
	values := s.configs(ctx)
	redirectURL := values["public.home.redirect_url"]
	// 配置了首页跳转时仍返回 HTML（而非服务端 302），
	// 以便先处理旧版 #base64 短码哈希（哈希不会到达服务端）。
	if redirectURL != "" {
		return s.renderHomeRedirect(ctx, redirectURL), http.StatusOK
	}
	view := s.view(ctx, "home", PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接服务正在运行",
		Message:  "这是短链接访问入口，请使用完整短链接访问目标内容。",
		Footer:   "GravityLink",
	})
	return s.render(view, http.StatusOK), http.StatusOK
}

// legacyHashScript 解析旧版引流宝 #base64 短码哈希并跳转。
// 例：/#TDlBa2Y= → base64 解码为 L9Akf → 跳转 /L9Akf。
const legacyHashScript = `<script>
(function(){
var h=location.hash.replace(/^#/,'');
if(!h)return;
try{
var pad=h.length%4;if(pad)h+=Array(5-pad).join('=');
h=h.replace(/-/g,'+').replace(/_/g,'/');
var code=decodeURIComponent(escape(atob(h)));
if(/^[A-Za-z0-9_-]{2,64}$/.test(code)){location.replace('/'+code);return;}
}catch(e){}
})();
</script>`

func (s *PublicPageService) renderHomeRedirect(ctx context.Context, redirectURL string) string {
	// 内联最小页：先尝试哈希短码，否则跳转配置的首页地址。
	// redirectURL 来自管理员配置，使用 html/template 转义。
	var buf bytes.Buffer
	tmpl := template.Must(template.New("homeRedirect").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.SiteName}}</title>
  <noscript><meta http-equiv="refresh" content="0;url={{.RedirectURL}}"></noscript>
</head>
<body>
` + legacyHashScript + `
<script>location.replace({{.RedirectURLJS}});</script>
</body>
</html>`))
	siteName := "GravityLink"
	if name := s.configValue(ctx, "public.site_name"); name != "" {
		siteName = name
	}
	_ = tmpl.Execute(&buf, map[string]string{
		"SiteName":      siteName,
		"RedirectURL":   redirectURL,
		"RedirectURLJS": strconv.Quote(redirectURL),
	})
	return buf.String()
}

func (s *PublicPageService) configValue(ctx context.Context, key string) string {
	var item model.SystemConfig
	if err := s.db.WithContext(ctx).Where("key_name = ?", key).First(&item).Error; err != nil {
		return ""
	}
	return item.Value
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
