package service

import (
	"context"
	"html"
	"net/http"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

type PublicPageService struct {
	db *gorm.DB
}

type PublicPageView struct {
	SiteName string
	Title    string
	Message  string
	Footer   string
}

func NewPublicPageService(db *gorm.DB) *PublicPageService {
	return &PublicPageService{db: db}
}

func (s *PublicPageService) Home(ctx context.Context) (string, int) {
	view := s.view(ctx, PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接服务正在运行",
		Message:  "这是短链接访问入口。请使用完整短链接访问目标内容。",
		Footer:   "GravityLink",
	})
	return renderPublicPage(view, http.StatusOK), http.StatusOK
}

func (s *PublicPageService) NotFound(ctx context.Context) (string, int) {
	view := s.view(ctx, PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接不存在或已失效",
		Message:  "请检查链接是否完整，或联系链接提供方确认当前链接状态。",
		Footer:   "GravityLink",
	})
	return renderPublicPage(view, http.StatusNotFound), http.StatusNotFound
}

func (s *PublicPageService) Gone(ctx context.Context) (string, int) {
	view := s.view(ctx, PublicPageView{
		SiteName: "GravityLink",
		Title:    "链接已过期",
		Message:  "该链接已超过有效期，无法继续访问。",
		Footer:   "GravityLink",
	})
	return renderPublicPage(view, http.StatusGone), http.StatusGone
}

func (s *PublicPageService) view(ctx context.Context, fallback PublicPageView) PublicPageView {
	values := s.configs(ctx)
	return PublicPageView{
		SiteName: firstNonEmpty(values["public.site_name"], fallback.SiteName),
		Title:    firstNonEmpty(values["public."+pageKey(fallback.Title)+".title"], fallback.Title),
		Message:  firstNonEmpty(values["public."+pageKey(fallback.Title)+".message"], fallback.Message),
		Footer:   firstNonEmpty(values["public.footer"], fallback.Footer),
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

func pageKey(title string) string {
	switch title {
	case "链接服务正在运行":
		return "home"
	case "链接已过期":
		return "gone"
	default:
		return "not_found"
	}
}

func renderPublicPage(view PublicPageView, status int) string {
	statusText := "HTTP " + http.StatusText(status)
	if status == http.StatusOK {
		statusText = "GravityLink"
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(view.Title) + `</title><style>
	:root{color:#172026;background:#f4f7f8;font-family:Inter,"Segoe UI","PingFang SC","Microsoft YaHei",Arial,sans-serif}
	*{box-sizing:border-box}body{min-height:100vh;margin:0;display:grid;place-items:center;background:#f4f7f8}
	main{width:min(92vw,520px);padding:34px;text-align:center;background:#fff;border:1px solid #dfe8eb;border-radius:8px;box-shadow:0 14px 34px rgba(18,35,44,.08)}
	.mark{width:48px;height:48px;margin:0 auto 18px;display:grid;place-items:center;border-radius:8px;background:#188d7c;color:#fff;font-weight:800}
	h1{margin:0 0 12px;font-size:26px;line-height:1.25}p{margin:0;color:#60747b;line-height:1.7}.status{margin-top:20px;color:#94a3aa;font-size:13px}footer{margin-top:18px;color:#94a3aa;font-size:13px}
	</style></head><body><main><div class="mark">G</div><h1>` + html.EscapeString(view.Title) + `</h1><p>` + html.EscapeString(view.Message) + `</p><div class="status">` + html.EscapeString(statusText) + `</div><footer>` + html.EscapeString(view.Footer) + `</footer></main></body></html>`
}
