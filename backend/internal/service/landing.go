package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

var (
	ErrLandingNotFound = errors.New("landing page not found")
	ErrInvalidTemplate = errors.New("invalid landing template")
)

const defaultThemeColor = "#0f766e"

type LandingService struct {
	db        *gorm.DB
	routing   *RoutingService
	templates *template.Template
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

type noticeContent struct {
	Message       string `json:"message"`
	ButtonText    string `json:"button_text"`
	Countdown     int    `json:"countdown"`
	ShowTargetURL bool   `json:"show_target_url"`
	ThemeColor    string `json:"theme_color"`
}

type customContent struct {
	HTML       string `json:"html"`
	ThemeColor string `json:"theme_color"`
}

func NewLandingService(db *gorm.DB, routing *RoutingService, templates *template.Template) *LandingService {
	return &LandingService{db: db, routing: routing, templates: templates}
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

// RenderByCode 渲染落地页 HTML，返回 (html, httpStatus, linkID, error)。
// linkID 用于调用方记录访问日志；渲染失败时返回 0。
func (s *LandingService) RenderByCode(ctx context.Context, code string) (string, int, uint64, error) {
	var link model.Link
	err := s.db.WithContext(ctx).Where("code = ?", code).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", http.StatusNotFound, 0, ErrLinkNotFound
	}
	if err != nil {
		return "", http.StatusInternalServerError, 0, err
	}
	if link.Status == model.LinkStatusDisabled {
		return "", http.StatusNotFound, 0, ErrLinkDisabled
	}
	if link.ExpireAt != nil && time.Now().After(*link.ExpireAt) {
		return "", http.StatusGone, 0, ErrLinkExpired
	}
	if link.LandingPageID == nil {
		return "", http.StatusNotFound, 0, ErrLandingNotFound
	}

	page, err := s.Get(ctx, *link.LandingPageID)
	if err != nil {
		return "", http.StatusNotFound, 0, err
	}

	switch page.Template {
	case "liveqr":
		target, err := s.routing.SelectTarget(ctx, link.ID)
		if err != nil {
			html, rerr := s.renderUnavailablePage(page.Title)
			return html, http.StatusOK, link.ID, rerr
		}
		html, rerr := s.renderLiveQRPage(page, target.TargetURL)
		return html, http.StatusOK, link.ID, rerr
	case "redirect_notice":
		html, rerr := s.renderNoticePage(page, derefString(link.TargetURL))
		return html, http.StatusOK, link.ID, rerr
	case "custom":
		html, rerr := s.renderCustomPage(page)
		return html, http.StatusOK, link.ID, rerr
	default:
		return "", http.StatusInternalServerError, 0, ErrInvalidTemplate
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

func (s *LandingService) renderLiveQRPage(page model.LandingPage, targetURL string) (string, error) {
	var content liveQRContent
	_ = json.Unmarshal(page.Content, &content)
	if content.Headline == "" {
		content.Headline = page.Title
	}
	if content.FooterText == "" {
		content.FooterText = "长按识别二维码"
	}
	if content.ThemeColor == "" {
		content.ThemeColor = defaultThemeColor
	}

	data := map[string]interface{}{
		"Title":      page.Title,
		"ThemeColor": content.ThemeColor,
		"Headline":   content.Headline,
		"Subtext":    content.Subtext,
		"FooterText": content.FooterText,
		"ShowLogo":   content.ShowLogo && content.LogoURL != "",
		"LogoURL":    content.LogoURL,
		"TargetURL":  targetURL,
		"IsImage":    isImageURL(targetURL),
		"ButtonText": "打开目标",
	}
	return s.executeTemplate("liveqr.html", data)
}

func (s *LandingService) renderNoticePage(page model.LandingPage, targetURL string) (string, error) {
	var content noticeContent
	_ = json.Unmarshal(page.Content, &content)
	if content.Message == "" {
		content.Message = "您即将前往外部网站，请注意安全。"
	}
	if content.ButtonText == "" {
		content.ButtonText = "继续访问"
	}
	if content.ThemeColor == "" {
		content.ThemeColor = defaultThemeColor
	}

	data := map[string]interface{}{
		"Title":         page.Title,
		"ThemeColor":    content.ThemeColor,
		"Message":       content.Message,
		"ButtonText":    content.ButtonText,
		"Countdown":     content.Countdown,
		"ShowTargetURL": content.ShowTargetURL,
		"TargetURL":     targetURL,
	}
	return s.executeTemplate("redirect_notice.html", data)
}

func (s *LandingService) renderCustomPage(page model.LandingPage) (string, error) {
	var content customContent
	_ = json.Unmarshal(page.Content, &content)
	if content.ThemeColor == "" {
		content.ThemeColor = defaultThemeColor
	}

	data := map[string]interface{}{
		"Title":      page.Title,
		"ThemeColor": content.ThemeColor,
		"HTML":       template.HTML(sanitizeCustomHTML(content.HTML)),
	}
	return s.executeTemplate("custom.html", data)
}

func (s *LandingService) renderUnavailablePage(title string) (string, error) {
	return s.executeTemplate("unavailable.html", map[string]interface{}{"Title": title})
}

// RenderTransitPage 渲染中转跳转页（meta refresh + 手动按钮）。
// 替换原先返回 JSON 的占位实现。
func (s *LandingService) RenderTransitPage(targetURL string) (string, error) {
	return s.executeTemplate("transit.html", map[string]interface{}{"TargetURL": targetURL})
}

func (s *LandingService) executeTemplate(name string, data interface{}) (string, error) {
	if s.templates == nil {
		return "", errors.New("templates not loaded")
	}
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func isImageURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".webp") || strings.HasSuffix(lower, ".gif")
}

// sanitizeCustomHTML 拦截最危险的注入向量：
//   - <script> 标签（大小写不敏感）
//   - javascript: 协议
//   - on* 事件属性（onerror、onclick 等）
//
// 这不是完整的 XSS 过滤器，仅满足「自定义落地页」的最低安全要求。
// 更复杂的 HTML 应通过 liveqr 或 redirect_notice 模板字段实现。
func sanitizeCustomHTML(raw string) string {
	lower := strings.ToLower(raw)
	out := raw

	// 拦截 <script 与 </script>（大小写不敏感，通过 lower 定位）
	for {
		idx := strings.Index(lower, "<script")
		if idx == -1 {
			break
		}
		out = out[:idx] + "&lt;script" + out[idx+len("<script"):]
		lower = lower[:idx] + "&lt;script" + lower[idx+len("<script"):]
	}
	for {
		idx := strings.Index(lower, "</script")
		if idx == -1 {
			break
		}
		out = out[:idx] + "&lt;/script" + out[idx+len("</script"):]
		lower = lower[:idx] + "&lt;/script" + lower[idx+len("</script"):]
	}
	for {
		idx := strings.Index(lower, "javascript:")
		if idx == -1 {
			break
		}
		out = out[:idx] + out[idx+len("javascript:"):]
		lower = lower[:idx] + lower[idx+len("javascript:"):]
	}

	return stripEventHandlers(out)
}

func stripEventHandlers(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(raw) {
		if raw[i] == '<' {
			end := strings.IndexByte(raw[i:], '>')
			if end == -1 {
				b.WriteString(raw[i:])
				break
			}
			tag := raw[i : i+end+1]
			b.WriteString(stripEventHandlersInTag(tag))
			i += end + 1
			continue
		}
		b.WriteByte(raw[i])
		i++
	}
	return b.String()
}

func stripEventHandlersInTag(tag string) string {
	lower := strings.ToLower(tag)
	var out strings.Builder
	out.Grow(len(tag))
	cursor := 0
	for cursor < len(tag) {
		idx := strings.Index(lower[cursor:], " on")
		if idx == -1 {
			out.WriteString(tag[cursor:])
			break
		}
		pos := cursor + idx
		j := pos + 2
		for j < len(tag) && (tag[j] >= 'a' && tag[j] <= 'z' || tag[j] >= 'A' && tag[j] <= 'Z' || tag[j] >= '0' && tag[j] <= '9' || tag[j] == '-' || tag[j] == '_') {
			j++
		}
		if j < len(tag) && tag[j] == '=' {
			k := j + 1
			if k < len(tag) && (tag[k] == '"' || tag[k] == '\'') {
				quote := tag[k]
				k++
				for k < len(tag) && tag[k] != quote {
					k++
				}
				if k < len(tag) {
					k++
				}
			} else {
				for k < len(tag) && tag[k] != ' ' && tag[k] != '>' {
					k++
				}
			}
			out.WriteString(tag[cursor:pos])
			cursor = k
			continue
		}
		out.WriteString(tag[cursor:j])
		cursor = j
	}
	return out.String()
}
