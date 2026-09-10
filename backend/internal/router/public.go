package router

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gravitylink/backend/internal/middleware"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
)

func registerPublicRoutes(engine *gin.Engine, deps Dependencies, domains *service.DomainCache, links *service.LinkService, landings *service.LandingService, pages *service.PublicPageService, recorder *service.AccessRecorder) {
	public := engine.Group("/")
	public.Use(middleware.HostRouter(domains, deps.Config, pages))
	public.GET("/", publicHome(pages))

	// 旧版 URL 兼容路由（必须在 /:code 之前注册）
	registerLegacyRoutes(public, deps, links, landings, pages, recorder)

	public.GET("/:code", dispatchByDomainType(deps, links, landings, pages, recorder))
	engine.NoRoute(func(c *gin.Context) {
		if isAPIPath(c.Request.URL.Path) {
			response.Error(c, http.StatusNotFound, 4004, "not found")
			return
		}
		html, status := pages.NotFound(c.Request.Context())
		c.Data(status, "text/html; charset=utf-8", []byte(html))
	})
}

// registerLegacyRoutes 注册旧版引流宝 URL 格式的兼容路由。
// 旧版格式：
//   /s/?key={code}                      短链/活码/渠道码统一入口
//   /s/dwz.php?key={code}               短链入口
//   /common/dwz/redirect/?key={code}    短链中转
//   /common/channel/redirect/?cid={id}  渠道码中转（数字ID）
//   /common/qun/redirect/?qid={id}      群活码中转（数字ID）
//   /common/shareCard/redirect/?sid={id} 分享卡片（数字ID）
//   /common/kf/redirect/?kid={id}       客服码中转（数字ID）
func registerLegacyRoutes(public *gin.RouterGroup, deps Dependencies, links *service.LinkService, landings *service.LandingService, pages *service.PublicPageService, recorder *service.AccessRecorder) {
	// /s/ 和 /s/dwz.php：按 code 查找并重定向
	// 同时支持 /s/?key=xxx（查询参数）和 /s/xxx（路径参数，nginx rewrite 等效）
	handleLegacyByCode := func(c *gin.Context) {
		code := strings.TrimSpace(c.Query("key"))
		if code == "" {
			code = strings.TrimSpace(c.Param("key"))
		}
		if code == "" {
			writeLegacyNotFound(c, pages)
			return
		}
		result, err := links.LegacyResolve(c.Request.Context(), code, false)
		if err != nil {
			writeResolveError(c, err, pages)
			return
		}
		legacyRedirect(c, deps, result, pages, recorder)
	}
	public.GET("/s/", handleLegacyByCode)
	public.GET("/s/dwz.php", handleLegacyByCode)
	public.GET("/s/:key", handleLegacyByCode)

	// /common/dwz/redirect/?key={code}：短链中转，按 code 查找
	public.GET("/common/dwz/redirect/", handleLegacyByCode)
	public.GET("/common/dwz/redirect/lx/", handleLegacyByCode) // 轮询域名格式

	// /common/channel/redirect/?cid={id}：渠道码，按 legacy_id 查找
	public.GET("/common/channel/redirect/", func(c *gin.Context) {
		legacyIDRedirect(c, deps, links, landings, pages, recorder, "cid")
	})

	// /common/qun/redirect/?qid={id}：群活码，按 legacy_id 查找
	public.GET("/common/qun/redirect/", func(c *gin.Context) {
		legacyIDRedirect(c, deps, links, landings, pages, recorder, "qid")
	})

	// /common/shareCard/redirect/?sid={id}：分享卡片展示页（扫码进入，配置微信 JS-SDK 引导分享）
	public.GET("/common/shareCard/redirect/", func(c *gin.Context) {
		legacyShareCardPage(c, deps, pages)
	})
	// /common/shareCard/redirect/signature：微信 JS-SDK 签名端点
	public.GET("/common/shareCard/redirect/signature", func(c *gin.Context) {
		legacyShareCardSignature(c, deps)
	})
	// /common/shareCard/?sid={id}：分享卡片落地页（被分享者打开，302 到目标）
	public.GET("/common/shareCard/", func(c *gin.Context) {
		legacyShareCardRedirect(c, deps, pages)
	})

	// /common/kf/redirect/?kid={id}：客服码，按 legacy_id 查找
	public.GET("/common/kf/redirect/", func(c *gin.Context) {
		legacyIDRedirect(c, deps, links, landings, pages, recorder, "kid")
	})

	// 旧版落地页格式（在落地域上直接访问）
	// /common/channel/?cid={id} → 渠道码落地（302 到目标）
	// /common/qun/?qid={id} → 群活码落地（渲染二维码页）
	// /common/kf/?kid={id} → 客服码落地（渲染客服页）
	// /common/dwz/?key={code} → 短链落地（302 到目标）
	// /common/shareCard/?sid={id} → 分享卡片落地（302 到目标）
	handleLegacyLanding := func(paramName string, byID bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			idStr := strings.TrimSpace(c.Query(paramName))
			if idStr == "" {
				writeLegacyNotFound(c, pages)
				return
			}
			// 先解析出链接 code，再走标准落地页渲染
			result, err := links.LegacyResolve(c.Request.Context(), idStr, byID)
			if err != nil {
				writeResolveError(c, err, pages)
				return
			}
			// 活码/kf：渲染落地页；短链/渠道：302
			if result.Link.Type == model.LinkTypeLiveQR {
				html, status, linkID, rerr := landings.RenderByCode(c.Request.Context(), result.Link.Code)
				if rerr != nil {
					if errors.Is(rerr, service.ErrNoRoutingTarget) {
						deps.Notifier.SendAsync("qr_exhausted:"+result.Link.Code, "⚠ 活码二维码已全部耗尽", "活码 "+result.Link.Code+" 的所有二维码达到阈值/到期/停用。")
					}
					writeResolveError(c, rerr, pages)
					return
				}
				if linkID > 0 {
					recorder.RecordAsync(service.AccessEvent{
						LinkID: linkID, IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
						Referer: c.Request.Referer(), VisitedAt: deps.Config.Now(),
					})
				}
				c.Header("Cache-Control", "no-store")
				c.Data(status, "text/html; charset=utf-8", []byte(html))
				return
			}
			// 短链/渠道：302 到目标
			legacyRedirect(c, deps, result, pages, recorder)
		}
	}
	public.GET("/common/channel/", handleLegacyLanding("cid", true))
	public.GET("/common/qun/", handleLegacyLanding("qid", true))
	public.GET("/common/kf/", handleLegacyLanding("kid", true))
	public.GET("/common/dwz/", handleLegacyLanding("key", false))
	// /common/shareCard/ 已在上方单独注册（legacyShareCardRedirect）
}

// legacyIDRedirect 按旧版数字 ID 查找链接并重定向。
func legacyIDRedirect(c *gin.Context, deps Dependencies, links *service.LinkService, landings *service.LandingService, pages *service.PublicPageService, recorder *service.AccessRecorder, paramName string) {
	idStr := strings.TrimSpace(c.Query(paramName))
	if idStr == "" {
		writeLegacyNotFound(c, pages)
		return
	}
	result, err := links.LegacyResolve(c.Request.Context(), idStr, true)
	if err != nil {
		writeResolveError(c, err, pages)
		return
	}
	legacyRedirect(c, deps, result, pages, recorder)
}

// legacyRedirect 执行旧版 URL 的重定向逻辑。
func legacyRedirect(c *gin.Context, deps Dependencies, result service.ResolveResult, pages *service.PublicPageService, recorder *service.AccessRecorder) {
	// UA 限制检查
	if allowed, hint := service.CheckAccessRule(c.Request.UserAgent(), result.Link.AccessRule); !allowed {
		html, status := pages.AccessDenied(c.Request.Context(), hint)
		c.Data(status, "text/html; charset=utf-8", []byte(html))
		return
	}
	// 活码只由落地域渲染时记一次访问日志
	if result.Link.Type != model.LinkTypeLiveQR {
		recorder.RecordAsync(service.AccessEvent{
			LinkID:     result.Link.ID,
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Referer:    c.Request.Referer(),
			VisitedAt:  deps.Config.Now(),
			ViaTransit: false,
		})
	}
	c.Redirect(http.StatusMovedPermanently, result.TargetURL)
}

func writeLegacyNotFound(c *gin.Context, pages *service.PublicPageService) {
	html, status := pages.NotFound(c.Request.Context())
	c.Data(status, "text/html; charset=utf-8", []byte(html))
}

func publicHome(pages *service.PublicPageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		html, status, redirectURL := pages.Home(c.Request.Context())
		if redirectURL != "" {
			c.Redirect(http.StatusFound, redirectURL)
			return
		}
		c.Data(status, "text/html; charset=utf-8", []byte(html))
	}
}

func dispatchByDomainType(deps Dependencies, links *service.LinkService, landings *service.LandingService, pages *service.PublicPageService, recorder *service.AccessRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(middleware.ContextDomainKey)
		if !exists {
			response.Error(c, http.StatusNotFound, 4101, "domain not found")
			return
		}

		domain, ok := value.(model.Domain)
		if !ok {
			response.Error(c, http.StatusInternalServerError, 5000, "domain context invalid")
			return
		}

		code := c.Param("code")
		switch domain.Type {
		case model.DomainTypeEntry:
			result, err := links.Resolve(c.Request.Context(), code)
			if err != nil {
				writeResolveError(c, err, pages)
				return
			}
			// P1：UA 访问限制检查（none 时放行；不满足时返回 403 引导页）
			if allowed, hint := service.CheckAccessRule(c.Request.UserAgent(), result.Link.AccessRule); !allowed {
				html, status := pages.AccessDenied(c.Request.Context(), hint)
				c.Data(status, "text/html; charset=utf-8", []byte(html))
				return
			}
			// 活码只由落地域渲染时记一次访问日志（此处 302 若也记则 PV 双计）。
			if result.Link.Type != model.LinkTypeLiveQR {
				recorder.RecordAsync(service.AccessEvent{
					LinkID:     result.Link.ID,
					IP:         c.ClientIP(),
					UserAgent:  c.Request.UserAgent(),
					Referer:    c.Request.Referer(),
					VisitedAt:  deps.Config.Now(),
					ViaTransit: false,
				})
			}
			c.Redirect(result.Status, result.TargetURL)
		case model.DomainTypeTransit:
			// 中转域：解析目标后渲染中转页（不再返回 JSON 占位）。
			result, err := links.Resolve(c.Request.Context(), code)
			if err != nil {
				writeResolveError(c, err, pages)
				return
			}
			recorder.RecordAsync(service.AccessEvent{
				LinkID:     result.Link.ID,
				IP:         c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Referer:    c.Request.Referer(),
				VisitedAt:  deps.Config.Now(),
				ViaTransit: true,
			})
			html, err := landings.RenderTransitPage(result.TargetURL)
			if err != nil {
				deps.Logger.Error("render transit page failed", "error", err)
				response.Error(c, http.StatusInternalServerError, 5000, "render transit page failed")
				return
			}
			c.Header("Cache-Control", "no-store")
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		case model.DomainTypeLanding:
			html, status, linkID, err := landings.RenderByCode(c.Request.Context(), code)
			if err != nil {
				// P1：活码全部二维码耗尽时告警运营（Notifier 内置 1 小时防抖）
				if errors.Is(err, service.ErrNoRoutingTarget) {
					deps.Notifier.SendAsync("qr_exhausted:"+code, "⚠ 活码二维码已全部耗尽", "活码 "+code+" 的所有二维码达到阈值/到期/停用，访客正在看到「暂无可用群」页面，请尽快补充二维码。")
				}
				writeResolveError(c, err, pages)
				return
			}
			if linkID > 0 {
				recorder.RecordAsync(service.AccessEvent{
					LinkID:     linkID,
					IP:         c.ClientIP(),
					UserAgent:  c.Request.UserAgent(),
					Referer:    c.Request.Referer(),
					VisitedAt:  deps.Config.Now(),
					ViaTransit: false,
				})
			}
			c.Header("Cache-Control", "no-store")
			c.Data(status, "text/html; charset=utf-8", []byte(html))
		default:
			deps.Logger.Warn("unsupported domain type", "host", domain.Host, "type", domain.Type)
			response.Error(c, http.StatusNotFound, 4101, "domain type unsupported")
		}
	}
}

func writeResolveError(c *gin.Context, err error, pages *service.PublicPageService) {
	switch {
	case errors.Is(err, service.ErrLinkNotFound), errors.Is(err, service.ErrLinkDisabled):
		html, status := pages.NotFound(c.Request.Context())
		c.Data(status, "text/html; charset=utf-8", []byte(html))
	case errors.Is(err, service.ErrLinkExpired):
		html, status := pages.Gone(c.Request.Context())
		c.Data(status, "text/html; charset=utf-8", []byte(html))
	case errors.Is(err, service.ErrUnsupportedLink), errors.Is(err, service.ErrTargetUnavailable):
		html, status := pages.NotFound(c.Request.Context())
		c.Data(status, "text/html; charset=utf-8", []byte(html))
	default:
		response.Error(c, http.StatusInternalServerError, 5000, "resolve link failed")
	}
}

func isAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

// legacyShareCardRedirect 旧版分享卡片落地页：/common/shareCard/?sid={id}
// 被分享者打开后 302 跳转到目标 URL。
func legacyShareCardRedirect(c *gin.Context, deps Dependencies, pages *service.PublicPageService) {
	card, ok := findShareCardByLegacyID(c, deps, pages)
	if !ok {
		return
	}
	if card.TargetURL == "" {
		writeLegacyNotFound(c, pages)
		return
	}
	_ = deps.DB.Model(&card).UpdateColumn("visits", gorm.Expr("visits + 1")).Error
	c.Redirect(http.StatusMovedPermanently, card.TargetURL)
}

// legacyShareCardPage 旧版分享卡片展示页：/common/shareCard/redirect/?sid={id}
// 扫码进入，展示卡片信息并配置微信 JS-SDK 引导分享。
// 分享出去的链接指向 /common/shareCard/?sid=xxx（被分享者打开后直接 302 跳转目标）。
func legacyShareCardPage(c *gin.Context, deps Dependencies, pages *service.PublicPageService) {
	card, ok := findShareCardByLegacyID(c, deps, pages)
	if !ok {
		return
	}
	// 分享链接指向跳转页，被分享者打开后直接 302 到目标
	sid := c.Query("sid")
	shareLink := "/common/shareCard/?sid=" + url.QueryEscape(sid)
	data := map[string]interface{}{
		"Title":       card.Title,
		"Description": card.Description,
		"ImageURL":    card.ImageURL,
		"TargetURL":   card.TargetURL,
		"ShareLink":   shareLink,
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = legacyShareTemplate.Execute(c.Writer, data)
}

// legacyShareTemplate 旧版分享卡片展示页模板。
// 与新版 /card/:id 模板的区别：分享链接用 ShareLink 字段（指向跳转页），而非 location.href。
var legacyShareTemplate = template.Must(template.New("legacyShare").Parse(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><style>body{font-family:system-ui;background:#eff5ff;color:#17243b;padding:24px}main{max-width:480px;margin:8vh auto;background:white;border-radius:24px;padding:32px}img{width:100px;height:100px;object-fit:cover;border-radius:16px}p{line-height:1.7;color:#64748b}a{display:block;background:#2879f8;color:white;padding:14px;text-align:center;border-radius:12px;text-decoration:none}</style><main><img src="{{.ImageURL}}" alt="卡片封面"><h1>{{.Title}}</h1><p>{{.Description}}</p><a href="{{.TargetURL}}" rel="noopener noreferrer">查看内容</a><p id="status">在微信中打开后，可通过右上角菜单分享。</p></main><script src="https://res.wx.qq.com/open/js/jweixin-1.6.0.js"></script><script>
const data={title:{{printf "%q" .Title}},desc:{{printf "%q" .Description}},imgUrl:{{printf "%q" .ImageURL}},link:location.origin+{{printf "%q" .ShareLink}}};
if(/MicroMessenger/i.test(navigator.userAgent)) fetch('/common/shareCard/redirect/signature?url='+encodeURIComponent(data.link)).then(r=>r.json()).then(r=>{if(r.code!==0)throw Error(r.message);wx.config({...r.data,debug:false,jsApiList:['updateAppMessageShareData','updateTimelineShareData']});wx.ready(()=>{wx.updateAppMessageShareData(data);wx.updateTimelineShareData(data);document.getElementById('status').textContent='请点击右上角菜单，分享给朋友或朋友圈。'});wx.error(()=>document.getElementById('status').textContent='分享配置失败，请联系管理员检查安全域名。')}).catch(()=>document.getElementById('status').textContent='微信分享暂不可用，请联系管理员检查公众号配置。');
</script></html>`))

func findShareCardByLegacyID(c *gin.Context, deps Dependencies, pages *service.PublicPageService) (model.ShareCard, bool) {
	sidStr := strings.TrimSpace(c.Query("sid"))
	if sidStr == "" {
		writeLegacyNotFound(c, pages)
		return model.ShareCard{}, false
	}
	sid, err := strconv.ParseUint(sidStr, 10, 64)
	if err != nil {
		writeLegacyNotFound(c, pages)
		return model.ShareCard{}, false
	}
	var card model.ShareCard
	if err := deps.DB.Where("legacy_id = ?", sid).First(&card).Error; err != nil {
		if err2 := deps.DB.Where("id = ?", sid).First(&card).Error; err2 != nil {
			writeLegacyNotFound(c, pages)
			return model.ShareCard{}, false
		}
	}
	if card.Status != "" && card.Status != "active" {
		writeLegacyNotFound(c, pages)
		return model.ShareCard{}, false
	}
	return card, true
}

// legacyShareCardSignature 微信 JS-SDK 签名端点（旧版分享卡片路由用）。
func legacyShareCardSignature(c *gin.Context, deps Dependencies) {
	pageURL := c.Query("url")
	if pageURL == "" {
		response.Error(c, 400, 4001, "missing url")
		return
	}
	wx := service.NewWechatService(deps.DB, filepath.Dir(deps.Config.ConfigFile))
	signature, err := wx.Sign(c.Request.Context(), pageURL)
	if err != nil {
		response.Error(c, 503, 5000, "wechat sign unavailable")
		return
	}
	c.Header("Cache-Control", "no-store")
	response.OK(c, signature)
}
