package service

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var (
	ErrLandingNotFound = errors.New("landing page not found")
	ErrInvalidTemplate = errors.New("invalid landing template")
)

type LandingService struct {
	db      *gorm.DB
	routing *RoutingService
}

type LandingInput struct {
	Template  string          `json:"template"`
	Title     string          `json:"title"`
	Content   json.RawMessage `json:"content"`
	DomainID  uint64          `json:"domain_id"`
	CreatedBy uint64          `json:"-"`
}

type liveQRContent struct {
	Headline   string `json:"headline"`
	Subtext    string `json:"subtext"`
	FooterText string `json:"footer_text"`
	ThemeColor string `json:"theme_color"`
	ShowLogo   bool   `json:"show_logo"`
	LogoURL    string `json:"logo_url"`
}

func NewLandingService(db *gorm.DB, routing *RoutingService) *LandingService {
	return &LandingService{db: db, routing: routing}
}

func (s *LandingService) List(ctx context.Context) ([]model.LandingPage, error) {
	var pages []model.LandingPage
	return pages, s.db.WithContext(ctx).Order("id DESC").Find(&pages).Error
}

func (s *LandingService) Create(ctx context.Context, input LandingInput) (model.LandingPage, error) {
	page, err := landingFromInput(input)
	if err != nil {
		return model.LandingPage{}, err
	}
	if err := s.db.WithContext(ctx).Create(&page).Error; err != nil {
		return model.LandingPage{}, err
	}
	return page, nil
}

func (s *LandingService) Get(ctx context.Context, id uint64) (model.LandingPage, error) {
	var page model.LandingPage
	err := s.db.WithContext(ctx).First(&page, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.LandingPage{}, ErrLandingNotFound
	}
	return page, err
}

func (s *LandingService) RenderByCode(ctx context.Context, code string) (string, int, error) {
	var link model.Link
	err := s.db.WithContext(ctx).Where("code = ?", code).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", http.StatusNotFound, ErrLinkNotFound
	}
	if err != nil {
		return "", http.StatusInternalServerError, err
	}
	if link.LandingPageID == nil {
		return "", http.StatusNotFound, ErrLandingNotFound
	}

	page, err := s.Get(ctx, *link.LandingPageID)
	if err != nil {
		return "", http.StatusNotFound, err
	}

	switch page.Template {
	case "liveqr":
		target, err := s.routing.SelectTarget(ctx, link.ID)
		if err != nil {
			return renderUnavailablePage(page.Title), http.StatusOK, nil
		}
		return renderLiveQRPage(page, target.TargetURL), http.StatusOK, nil
	case "redirect_notice":
		return renderNoticePage(page, derefString(link.TargetURL)), http.StatusOK, nil
	case "custom":
		return renderCustomPage(page), http.StatusOK, nil
	default:
		return "", http.StatusInternalServerError, ErrInvalidTemplate
	}
}

func landingFromInput(input LandingInput) (model.LandingPage, error) {
	if !validTemplate(input.Template) {
		return model.LandingPage{}, ErrInvalidTemplate
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "GravityLink"
	}
	content := input.Content
	if len(content) == 0 {
		content = json.RawMessage(`{}`)
	}
	return model.LandingPage{
		Template:  input.Template,
		Title:     title,
		Content:   content,
		DomainID:  input.DomainID,
		CreatedBy: input.CreatedBy,
	}, nil
}

func validTemplate(template string) bool {
	switch template {
	case "liveqr", "redirect_notice", "custom":
		return true
	default:
		return false
	}
}

func renderLiveQRPage(page model.LandingPage, targetURL string) string {
	var content liveQRContent
	_ = json.Unmarshal(page.Content, &content)
	if content.Headline == "" {
		content.Headline = page.Title
	}
	if content.FooterText == "" {
		content.FooterText = "长按识别二维码"
	}
	if content.ThemeColor == "" {
		content.ThemeColor = "#1677ff"
	}

	target := html.EscapeString(targetURL)
	media := `<a class="button" href="` + target + `">打开目标</a>`
	if isImageURL(targetURL) {
		media = `<img class="qr" src="` + target + `" alt="二维码" />`
	}

	return baseHTML(page.Title, content.ThemeColor, `
		<h1>`+html.EscapeString(content.Headline)+`</h1>
		<p>`+html.EscapeString(content.Subtext)+`</p>
		`+media+`
		<footer>`+html.EscapeString(content.FooterText)+`</footer>
	`)
}

func renderNoticePage(page model.LandingPage, targetURL string) string {
	return baseHTML(page.Title, "#1677ff", `
		<h1>`+html.EscapeString(page.Title)+`</h1>
		<p>您即将前往外部网站，请注意安全。</p>
		<a class="button" href="`+html.EscapeString(targetURL)+`">继续访问</a>
	`)
}

func renderCustomPage(page model.LandingPage) string {
	type customContent struct {
		HTML string `json:"html"`
	}
	var content customContent
	_ = json.Unmarshal(page.Content, &content)
	return baseHTML(page.Title, "#1677ff", sanitizeCustomHTML(content.HTML))
}

func renderUnavailablePage(title string) string {
	return baseHTML(title, "#8a94a6", `<h1>暂无可用群</h1><p>请稍后再试。</p>`)
}

func baseHTML(title string, themeColor string, body string) string {
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(title) + `</title><style>
	body{margin:0;min-height:100vh;display:grid;place-items:center;background:#f5f8fa;color:#172026;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif}
	main{width:min(92vw,420px);text-align:center;background:#fff;border:1px solid #dde8ec;border-radius:8px;padding:28px;box-shadow:0 10px 28px rgba(20,35,45,.08)}
	h1{font-size:24px;margin:0 0 10px}p{color:#5f7178;margin:0 0 18px}.qr{width:min(72vw,260px);height:auto;border-radius:8px}.button{display:inline-flex;min-height:42px;align-items:center;justify-content:center;padding:0 18px;border-radius:6px;background:` + html.EscapeString(themeColor) + `;color:#fff;text-decoration:none}footer{margin-top:16px;color:#73858c;font-size:14px}
	</style></head><body><main>` + body + `</main></body></html>`
}

func isImageURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".webp") || strings.HasSuffix(lower, ".gif")
}

func sanitizeCustomHTML(raw string) string {
	raw = strings.ReplaceAll(raw, "<script", "&lt;script")
	raw = strings.ReplaceAll(raw, "</script>", "&lt;/script&gt;")
	raw = strings.ReplaceAll(raw, "javascript:", "")
	return raw
}
