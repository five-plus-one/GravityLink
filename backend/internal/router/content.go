package router

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gravitylink/backend/internal/model"
	"gravitylink/backend/internal/response"
	"gravitylink/backend/internal/service"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func httpURL(value string) bool {
	u, e := url.Parse(value)
	return e == nil && u.Host != "" && u.User == nil && (u.Scheme == "http" || u.Scheme == "https")
}

func registerContentRoutes(engine *gin.Engine, admin *gin.RouterGroup, deps Dependencies, domains *service.DomainCache) {
	dir := filepath.Dir(deps.Config.ConfigFile)
	uploads := filepath.Join(dir, "uploads")
	wx := service.NewWechatService(deps.DB, dir)
	fail := func(c *gin.Context) { response.Error(c, 400, 4001, "配置无效或资源不可用") }
	admin.GET("/wechat-config", func(c *gin.Context) {
		id, set, e := wx.Status(c.Request.Context())
		if e != nil {
			fail(c)
			return
		}
		response.OK(c, gin.H{"appid": id, "secret_configured": set})
	})
	admin.PUT("/wechat-config", func(c *gin.Context) {
		var in struct {
			AppID  string `json:"appid"`
			Secret string `json:"secret"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c)
			return
		}
		if e := wx.Save(c.Request.Context(), strings.TrimSpace(in.AppID), strings.TrimSpace(in.Secret)); e != nil {
			fail(c)
			return
		}
		response.OK(c, gin.H{"saved": true})
	})
	admin.POST("/wechat-config/check", func(c *gin.Context) {
		// Fixed URL: this checks ticket availability, not a client or domain.
		_, err := wx.Sign(c.Request.Context(), "https://localhost/wechat-check")
		if err != nil {
			stage, code := service.WechatDiagnostic(err)
			response.OK(c, gin.H{"ok": false, "stage": stage, "wechat_code": code})
			return
		}
		response.OK(c, gin.H{"ok": true, "stage": "jsapi_ticket", "wechat_code": 0})
	})
	admin.GET("/materials", func(c *gin.Context) {
		var items []model.Material
		if deps.DB.Order("id DESC").Find(&items).Error != nil {
			fail(c)
			return
		}
		response.OK(c, gin.H{"items": items})
	})
	admin.POST("/materials", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6<<20)
		f, _, e := c.Request.FormFile("file")
		if e != nil {
			fail(c)
			return
		}
		defer f.Close()
		if c.Request.MultipartForm != nil {
			defer c.Request.MultipartForm.RemoveAll()
		}
		b, e := io.ReadAll(io.LimitReader(f, (5<<20)+1))
		if e != nil || len(b) > 5<<20 {
			fail(c)
			return
		}
		ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}[http.DetectContentType(b)]
		if ext == "" {
			fail(c)
			return
		}
		id := make([]byte, 16)
		if _, e = rand.Read(id); e != nil {
			fail(c)
			return
		}
		name := hex.EncodeToString(id) + ext
		if os.MkdirAll(uploads, 0700) != nil {
			fail(c)
			return
		}
		if os.WriteFile(filepath.Join(uploads, name), b, 0600) != nil {
			fail(c)
			return
		}
		item := model.Material{Name: name, Path: "/uploads/" + name}
		if deps.DB.Create(&item).Error != nil {
			fail(c)
			return
		}
		response.OK(c, item)
	})
	engine.GET("/uploads/:name", func(c *gin.Context) {
		name := c.Param("name")
		if filepath.Base(name) != name || strings.Contains(name, "..") {
			c.Status(404)
			return
		}
		var item model.Material
		if deps.DB.Where("path = ?", "/uploads/"+name).First(&item).Error != nil {
			c.Status(404)
			return
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "public, max-age=86400")
		c.File(filepath.Join(uploads, name))
	})
	admin.GET("/share-cards", func(c *gin.Context) {
		var items []model.ShareCard
		if deps.DB.Order("id DESC").Find(&items).Error != nil {
			fail(c)
			return
		}
		for i := range items {
			var d model.Domain
			if deps.DB.First(&d, items[i].DomainID).Error == nil && d.Status == model.StatusActive {
				items[i].PublicURL = d.Scheme + "://" + d.Host + "/card/" + strconv.FormatUint(items[i].ID, 10)
			}
		}
		response.OK(c, gin.H{"items": items})
	})
	save := func(c *gin.Context) {
		var in struct {
			DomainID    uint64
			Title       string
			Description string
			ImageURL    string
			TargetURL   string
			Status      string
		}
		if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Title) == "" || len(in.Title) > 128 || len(in.Description) > 500 || !httpURL(in.TargetURL) || !httpURL(in.ImageURL) || (in.Status != "active" && in.Status != "disabled") {
			fail(c)
			return
		}
		var d model.Domain
		if deps.DB.Where("id = ? AND status = ? AND type = ?", in.DomainID, model.StatusActive, model.DomainTypeEntry).First(&d).Error != nil {
			fail(c)
			return
		}
		var item model.ShareCard
		if c.Param("id") != "" {
			id, ok := parseID(c)
			if !ok {
				return
			}
			if deps.DB.First(&item, id).Error != nil {
				fail(c)
				return
			}
		}
		item.DomainID = in.DomainID
		item.Title = in.Title
		item.Description = in.Description
		item.ImageURL = in.ImageURL
		item.TargetURL = in.TargetURL
		item.Status = in.Status
		var saveErr error
		if item.ID == 0 {
			saveErr = deps.DB.Create(&item).Error
		} else {
			saveErr = deps.DB.Model(&item).Updates(map[string]any{"domain_id": item.DomainID, "title": item.Title, "description": item.Description, "image_url": item.ImageURL, "target_url": item.TargetURL, "status": item.Status}).Error
		}
		if saveErr != nil {
			fail(c)
			return
		}
		response.OK(c, item)
	}
	admin.POST("/share-cards", save)
	admin.PUT("/share-cards/:id", save)
	getCard := func(c *gin.Context) (model.ShareCard, bool) {
		var card model.ShareCard
		id, e := strconv.ParseUint(c.Param("id"), 10, 64)
		d, ok := domains.Get(c.Request.Host)
		if e != nil || !ok || deps.DB.Where("id = ? AND domain_id = ? AND status = ?", id, d.ID, "active").First(&card).Error != nil {
			c.Status(404)
			return card, false
		}
		return card, true
	}
	engine.GET("/card/:id", func(c *gin.Context) {
		card, ok := getCard(c)
		if !ok {
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Header("Content-Type", "text/html; charset=utf-8")
		if deps.DB.Model(&card).UpdateColumn("visits", gorm.Expr("visits + 1")).Error != nil {
			c.Status(500)
			return
		}
		_ = shareTemplate.Execute(c.Writer, card)
	})
	// /card/:id/go 跳转端点：被分享者点击卡片后直接 302 到目标 URL
	engine.GET("/card/:id/go", func(c *gin.Context) {
		card, ok := getCard(c)
		if !ok {
			return
		}
		_ = deps.DB.Model(&card).UpdateColumn("visits", gorm.Expr("visits + 1")).Error
		c.Redirect(http.StatusFound, card.TargetURL)
	})
	engine.GET("/card/:id/signature", func(c *gin.Context) {
		card, ok := getCard(c)
		if !ok {
			return
		}
		d, _ := domains.Get(c.Request.Host)
		page := c.Query("url")
		u, e := url.Parse(page)
		cardPath := "/card/" + strconv.FormatUint(card.ID, 10)
		if e != nil || u.Scheme != d.Scheme || u.Host != d.Host || u.Path != cardPath || u.User != nil {
			fail(c)
			return
		}
		signature, e := wx.Sign(c.Request.Context(), page)
		if e != nil {
			response.Error(c, 503, 5000, service.ErrWechat.Error())
			return
		}
		c.Header("Cache-Control", "no-store")
		response.OK(c, signature)
	})
}

// jsString 将字符串转为安全的 JavaScript 字符串字面量（含引号）。
func jsString(s string) template.JS {
	b, _ := json.Marshal(s)
	return template.JS(b)
}

var shareTemplate = template.Must(template.New("share").Funcs(template.FuncMap{"js": jsString}).Parse(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><style>body{font-family:system-ui;background:#eff5ff;color:#17243b;padding:24px}main{max-width:480px;margin:8vh auto;background:white;border-radius:24px;padding:32px}img{width:100px;height:100px;object-fit:cover;border-radius:16px}p{line-height:1.7;color:#64748b}a{display:block;background:#2879f8;color:white;padding:14px;text-align:center;border-radius:12px;text-decoration:none}</style><main><img src="{{.ImageURL}}" alt="卡片封面"><h1>{{.Title}}</h1><p>{{.Description}}</p><a href="{{.TargetURL}}" rel="noopener noreferrer">查看内容</a><p id="status">在微信中打开后，可通过右上角菜单分享。</p></main><script src="https://res.wx.qq.com/open/js/jweixin-1.6.0.js"></script><script>
const pageUrl=location.href.split('#')[0];
const shareLink=location.origin+location.pathname+'/go';
const data={title:{{js .Title}},desc:{{js .Description}},imgUrl:{{js .ImageURL}},link:shareLink};
if(/MicroMessenger/i.test(navigator.userAgent)) fetch(location.pathname+'/signature?url='+encodeURIComponent(pageUrl)).then(r=>r.json()).then(r=>{if(r.code!==0)throw Error(r.message);wx.config({...r.data,debug:false,jsApiList:['updateAppMessageShareData','updateTimelineShareData']});wx.ready(()=>{wx.updateAppMessageShareData(data);wx.updateTimelineShareData(data);document.getElementById('status').textContent='请点击右上角菜单，分享给朋友或朋友圈。'});wx.error(()=>document.getElementById('status').textContent='分享配置失败，请联系管理员检查安全域名。')}).catch(()=>document.getElementById('status').textContent='微信分享暂不可用，请联系管理员检查公众号配置。');
</script></html>`))
