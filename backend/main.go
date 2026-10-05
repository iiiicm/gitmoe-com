package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"techlife-blog/database"
	"techlife-blog/handlers"
	"techlife-blog/middleware"
	"techlife-blog/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 数据库路径
	dbPath := "data/techlife.db"
	if envPath := os.Getenv("DB_PATH"); envPath != "" {
		dbPath = envPath
	}
	// 确保目录存在
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	// 初始化数据库
	if err := database.Init(dbPath); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	fmt.Println("✅ 数据库初始化成功:", dbPath)

	// 文章正文预缓存（P1/P2）：启动即预热最新发布的文章，降低爬虫/首发流量下的 DB 压力。
	handlers.WarmArticleCache(200)

	// Gin 模式
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORS())
	r.Use(middleware.DomainGuard())   // 域名白名单防镜像
	r.Use(middleware.CacheControl(0)) // 静态资源浏览器缓存

	// 静态文件服务（前端构建后的 dist 目录）
	distPath := "../frontend/dist"
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		distPath = "./static"
	}
	if _, err := os.Stat(distPath); err == nil {
		r.Static("/assets", filepath.Join(distPath, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))
		// index.html 不缓存，确保每次部署后浏览器能加载最新 JS
		r.GET("/index.html", func(c *gin.Context) {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Expires", "0")
			c.Header("Pragma", "no-cache")
			c.File(filepath.Join(distPath, "index.html"))
		})
		// ===== Vue 网页版部署在 /web 子目录（vite base=/web/，仅改写 URL 前缀，dist 目录结构不变）=====
		r.Static("/web/assets", filepath.Join(distPath, "assets"))
		r.Static("/web/ueditor", filepath.Join(distPath, "ueditor"))
		webIndexHandler := func(c *gin.Context) {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Expires", "0")
			c.Header("Pragma", "no-cache")
			c.File(filepath.Join(distPath, "index.html"))
		}
		r.GET("/web/index.html", webIndexHandler)
		r.GET("/web", webIndexHandler)

		// ===== client.gitmoe.com 根路径网页版（vite base=/，独立构建产物）=====
		// OpenResty client.gitmoe.com 的 location / 反代到 /clientroot/，
		// 由这里托管 base=/ 的 SPA（assets 路径为 /assets/...，与主站 /assets 同源复用）。
		// 未构建 static_root 时自动回退到 /web 那份（base=/web/，会 301 到 /web，见 client conf）。
		rootDist := "./static_root"
		if _, err := os.Stat(rootDist); err == nil {
			clientRootIndex := func(c *gin.Context) {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Expires", "0")
				c.Header("Pragma", "no-cache")
				c.File(filepath.Join(rootDist, "index.html"))
			}
			r.Static("/clientroot/assets", filepath.Join(rootDist, "assets"))
			r.Static("/clientroot/ueditor", filepath.Join(rootDist, "ueditor"))
			r.GET("/clientroot", clientRootIndex)
			r.GET("/clientroot/", clientRootIndex)
			r.GET("/clientroot/index.html", clientRootIndex)
			fmt.Println("✅ client.gitmoe.com 根网页版:", rootDist)
		}
		// SPA 回退：仅 /web 前缀的未知路径返回 Vue index.html（Vue base=/web/，保证客户端路由）。
		// 其余（/api、/assets、/ueditor 等）一律返回 404，禁止当 SPA 返回，
		// 否则浏览器加载 Vue 后会因 base=/web/ 把 /api/... 重定向到 /web/...（线上已出现的跳转 bug）。
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// 先检查是否是上传的文件
			if strings.HasPrefix(path, "/data/uploads/") {
				c.File("." + path)
				return
			}
			// 禁止网页访问客户端路由（/client/* 仅限桌面/Android 客户端）
			// 注意：/clientroot 是 client.gitmoe.com 根路径网页版托管前缀，必须放行。
			if strings.HasPrefix(path, "/client") && !strings.HasPrefix(path, "/clientroot") {
				c.Redirect(302, "/")
				return
			}
			// client.gitmoe.com 根网页版的 SPA 回退（base=/ 的独立构建）
			if strings.HasPrefix(path, "/clientroot") {
				if _, err := os.Stat("./static_root/index.html"); err == nil {
					c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
					c.Header("Expires", "0")
					c.Header("Pragma", "no-cache")
					c.File("./static_root/index.html")
					return
				}
			}
			// 仅 /web 路径走 Vue SPA 回退；其余返回 404，不发 SPA
			if strings.HasPrefix(path, "/web") {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Expires", "0")
				c.Header("Pragma", "no-cache")
				c.File(filepath.Join(distPath, "index.html"))
				return
			}
			c.JSON(404, gin.H{"code": 404, "message": "Not Found"})
		})
		fmt.Println("✅ 前端静态文件:", distPath)
	} else {
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/data/uploads/") {
				c.File("." + c.Request.URL.Path)
				return
			}
		})
	}

	// 上传文件静态服务
	os.MkdirAll("./data/uploads/covers", 0755)
	os.MkdirAll("./data/uploads/attachments", 0755)
	os.MkdirAll("./data/uploads/videos", 0755)
	os.MkdirAll("./data/uploads/comics", 0755)
	os.MkdirAll("./data/uploads/music", 0755)
	os.MkdirAll("./data/uploads/novels", 0755)
	r.Static("/data/uploads", "./data/uploads")

	// R2 流式回源（client.gitmoe.com / cdn.gitmoe.com 经 OpenResty 改写到 /r2/，由 storage 抽象决定走 R2 还是本地）。
	// 长期目标：用此接口替代 ./data/uploads/ 本地磁盘；保留 r.Static 作为旧域名与本地兜底。
	r.GET("/r2/*filepath", handlers.R2ServeFile)

	// 启动 P2P 中继缓冲清理器
	handlers.StartP2PCleaner()
	handlers.StartResourceP2PCleaner()

	// 启动百度网盘 Cookie 自动巡检（失效实时检测 + 通知）
	handlers.StartBaiduCookieWatcher()

	// 初始化 JWT 密钥（强制读取环境变量，杜绝硬编码弱密钥）
	utils.InitJWTSecret()

	// ============================================
	// 伪静态 SEO 页面（服务端渲染，轻量 HTML）
	// ============================================
	r.GET("/article/:id.html", handlers.ArticlePage)

	// ============================================
	// API 路由
	// ============================================
	api := r.Group("/api")
	{
		// 安装
		api.GET("/install/check", handlers.CheckInstall)
		api.POST("/install/run", handlers.RunInstall)

		// 认证（高风控接口叠加 IP 限流，防爆破 / 防刷，#227）
		api.POST("/auth/register", middleware.RateLimitPerIP(20, time.Minute), handlers.Register)
		api.POST("/auth/login", middleware.RateLimitPerIP(10, time.Minute), handlers.Login)
		api.POST("/auth/social", handlers.SocialLogin)
		api.POST("/auth/phone", handlers.PhoneLogin)
		api.POST("/auth/sms", handlers.SendSMS)
		api.POST("/auth/send-email-code", middleware.RateLimitPerIP(10, time.Minute), handlers.SendEmailCode)
		// 注册页实时校验邮箱是否已被注册
		api.POST("/auth/check-email", handlers.CheckEmail)
		// 邮箱验证码找回密码（「忘记密码」卡片）
		api.POST("/auth/reset-password", middleware.RateLimitPerIP(10, time.Minute), handlers.ResetPassword)
		// 扫码登录（网页 / Windows 桌面端）：公开创建与轮询接口（叠加 IP 限流，#227）
		api.POST("/auth/qr/create", middleware.RateLimitPerIP(20, time.Minute), handlers.QrCreate)
		api.GET("/auth/qr/:token/status", middleware.RateLimitPerIP(120, time.Minute), handlers.QrStatus)
		api.POST("/auth/email-login", handlers.EmailLogin)
		// 免密登录（验证码登录，账号不存在自动创建）
		api.POST("/auth/passwordless/send-code", handlers.PasswordlessSendCode)
		api.POST("/auth/passwordless/login", handlers.PasswordlessLogin)

		// 设备解绑（用户自助，72h 冷却 + IP 限流）
		api.POST("/unbind-device", middleware.RateLimitPerIP(5, time.Hour), handlers.UnbindDevice)

		// 文字点选验证码（AJ-Captcha clickWord 风格，注册验证用）
		api.GET("/captcha/get", middleware.RateLimitPerIP(60, time.Minute), handlers.CaptchaGet)
		api.POST("/captcha/check", middleware.RateLimitPerIP(30, time.Minute), handlers.CaptchaCheck)

		// GitHub OAuth
		api.GET("/auth/github/authorize", handlers.GitHubAuthorize)
		api.GET("/auth/github/callback", handlers.GitHubCallback)

		// 公开接口
		api.GET("/articles", middleware.OptionalAuth(), handlers.GetArticles)
		api.GET("/articles/:id", middleware.OptionalAuth(), handlers.GetArticle)
		api.GET("/articles/:id/images", handlers.GetArticleImages)
		api.GET("/tags", handlers.GetTags)
		api.GET("/categories", handlers.GetCategories)
		api.GET("/payment/config", handlers.GetPaymentConfig)
		api.GET("/memberships", handlers.GetMemberships)
		// 新会员体系配置（1.0.5 客户端拉取；旧 /memberships 继续服务老客户端）
		api.GET("/memberships-v2/config", handlers.GetMembershipV2Config)
		api.GET("/navigations", handlers.GetNavigations)
		api.GET("/content/chapters/:contentId", handlers.GetChapters)
		api.GET("/chapters/:id", middleware.OptionalAuth(), handlers.GetChapter)
		api.GET("/video/parse", handlers.ParseXVideo)
		api.GET("/sitemap.xml", handlers.GenerateSitemap)
		api.GET("/sitemap-shard.xml", handlers.GenerateSitemapShard)
		api.GET("/robots.txt", handlers.RobotsTxt)
		api.GET("/llms.txt", handlers.LlmsTxt)
		api.GET("/feed.xml", handlers.FeedXml)

		// 爬虫访问记录（公开，由 OpenResty 采集调用）
		api.POST("/crawler-track", handlers.TrackCrawler)
		api.GET("/crawler-blocklist", handlers.CrawlerBlocklist)

		// 设置（公开）
		api.GET("/settings", func(c *gin.Context) {
			siteName := database.GetSetting("site_name")
			siteSubtitle := database.GetSetting("site_subtitle")
			siteDescription := database.GetSetting("site_description")
			siteOwner := database.GetSetting("site_owner")
			siteEmail := database.GetSetting("site_email")
			siteGithub := database.GetSetting("site_github")
			siteWechat := database.GetSetting("site_wechat")
			pageSize := database.GetSetting("page_size")
			registerEnabled := database.GetSetting("register_enabled")
			allowedDomains := database.GetSetting("allowed_domains")
			seoTitle := database.GetSetting("seo_title")
			seoDesc := database.GetSetting("seo_description")
			seoKeywords := database.GetSetting("seo_keywords")
			thumbnailHeight := database.GetSetting("thumbnail_height")
			videoAdEnabled := database.GetSetting("video_ad_enabled")
			videoAdUrl := database.GetSetting("video_ad_url")
			siteLogo := database.GetSetting("site_logo")
			contentNavEnabled := database.GetSetting("content_nav_enabled")
			contentNavItems := database.GetSetting("content_nav_items")
			registerEmailMode := database.GetSetting("register_email_mode")
			registerPhoneMode := database.GetSetting("register_phone_mode")
			footerCode := database.GetSetting("footer_code")

			// 下载相关配置（后台可配，前端付费面板读取，不再硬编码）
			dailyFreeBase := database.GetSetting("daily_free_base")
			if dailyFreeBase == "" {
				dailyFreeBase = "1"
			}
			dailyFreeSigninBonus := database.GetSetting("daily_free_signin_bonus")
			if dailyFreeSigninBonus == "" {
				dailyFreeSigninBonus = "1"
			}

			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"siteName":             siteName,
					"siteSubtitle":         siteSubtitle,
					"siteDescription":      siteDescription,
					"siteOwner":            siteOwner,
					"siteEmail":            siteEmail,
					"siteGithub":           siteGithub,
					"siteWechat":           siteWechat,
					"siteLogo":             siteLogo,
					"pageSize":             pageSize,
					"registerEnabled":      registerEnabled,
					"registerEmailMode":    registerEmailMode,
					"registerPhoneMode":    registerPhoneMode,
					"contentNavEnabled":    contentNavEnabled,
					"contentNavItems":      contentNavItems,
					"allowedDomains":       allowedDomains,
					"seoTitle":             seoTitle,
					"seoDescription":       seoDesc,
					"seoKeywords":          seoKeywords,
					"thumbnailHeight":      thumbnailHeight,
					"videoAdEnabled":       videoAdEnabled,
					"videoAdUrl":           videoAdUrl,
					"footerCode":           footerCode,
					"dailyFreeBase":        dailyFreeBase,
					"dailyFreeSigninBonus": dailyFreeSigninBonus,
				},
			})
		})

		// 内容版本号（前端轮询实时刷新用）
		api.GET("/content-version", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "data": gin.H{"version": database.GetContentVersion()}})
		})

		// APP 热更新信息（公开）
		api.GET("/app-update", func(c *gin.Context) {
			enabled := database.GetSetting("app_update_enabled") == "true"
			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"enabled": enabled,
					"version": database.GetSetting("app_update_version"),
					"url":     database.GetSetting("app_update_url"),
					"content": database.GetSetting("app_update_content"),
				},
			})
		})

		// Banner 和首页模块（公开）
		api.GET("/banners", handlers.GetBanners)
		api.GET("/home-modules", handlers.GetHomeModules)

		// 客户端配置（公开，由登录窗口 / 桌面端调用）
		api.GET("/client-config", handlers.GetClientConfig)
		// 安卓端配置（公开，由原生 Android App 启动后调用）
		api.GET("/android-config", handlers.GetAndroidConfig)
		// 网站端配置（公开，由 web-next 调用；消除孤儿端点）
		api.GET("/web-config", handlers.GetWebConfig)

		// 短剧 P2P / PCDN 信令（公开，无需登录：只交换资源 key 与节点地址）
		api.POST("/pcdn/announce", handlers.PcdnAnnounce)
		api.GET("/pcdn/peers", handlers.PcdnPeers)

		// 红包雨配置（公开：首页轮询判断是否处于时段，无需登录）
		api.GET("/redpacket/config", handlers.GetRedPacketConfig)

		// 首页 Banner（公开，按 device 返回电脑/手机首页 banner）
		api.GET("/home-banner", handlers.GetHomeBanner)

		// 首页分类目录（公开，按 device 返回有序分类 ID 数组）
		api.GET("/home-categories", handlers.GetHomeCategories)

		// 首页聚合配置（公开，一次性返回 banner + 展示配置 + 分类目录，按 device）
		api.GET("/home-config", handlers.GetHomeConfig)

		// 客户端左侧支付成功弹幕墙（公开，虚拟+真实订单混合）
		api.GET("/client/payment-barrage", handlers.GetPaymentBarrage)
		// 用户弹幕列表（公开，用于弹幕墙展示）
		api.GET("/barrage/list", handlers.ListBarrages)

		// 需要登录的接口
		auth := api.Group("")
		auth.Use(middleware.Auth(), middleware.RecordActiveSource())
		{
			// 用户
			auth.GET("/user/current", handlers.GetCurrentUser)
			auth.PUT("/user/profile", handlers.UpdateProfile)
			auth.PUT("/user/username", handlers.UpdateUsername)
			// 兼容别名：旧版客户端误把改用户名接口写成 /auth/update-username（导致强制改名提交 404）。
			// 保留此别名可让未升级的旧客户端立即恢复正常，待下次客户端发版后可移除。
			auth.PUT("/auth/update-username", handlers.UpdateUsername)
			auth.POST("/upload-avatar", handlers.UploadAvatar)
			auth.PUT("/user/password", handlers.ChangePassword)
			auth.PUT("/user/email", handlers.ChangeEmail)
			auth.GET("/user/likes", handlers.GetUserLikes)
			auth.GET("/user/comments", handlers.GetUserComments)
			// 扫码登录：App 扫码并确认（需登录，且须为扫码者本人）
			auth.POST("/auth/qr/:token/scan", handlers.QrScan)
			auth.POST("/auth/qr/:token/confirm", handlers.QrConfirm)
			auth.GET("/user/statistics", handlers.GetStatistics)

			// GitHub OAuth（需要登录）
			auth.GET("/auth/github/status", handlers.GitHubBindStatus)
			auth.POST("/auth/github/unbind", handlers.GitHubUnbind)

			// 签到
			auth.POST("/checkin", handlers.CheckIn)
			auth.GET("/checkin/history", handlers.GetCheckInHistory)

			// 弹幕：发送（登录后）+ 列表（公开）
			auth.POST("/barrage/send", handlers.SendBarrage)

			// 消息
			auth.GET("/messages", handlers.GetMessages)
			auth.PUT("/messages/:id/read", handlers.ReadMessage)
			auth.PUT("/messages/read-all", handlers.ReadAllMessages)
			auth.DELETE("/messages/:id", handlers.DeleteMessage)

			// 徽章
			auth.GET("/badges", handlers.GetBadges)

			// 发布
			auth.POST("/articles/publish", handlers.PublishArticle)

			// ===== 客户端本地图库（缓存/去重/还原） =====
			auth.POST("/cache/claim", handlers.ClaimArticle)
			auth.GET("/cache/suggest", handlers.SuggestArticles)
			auth.POST("/articles/:id/restore-image", handlers.RestoreArticleImage)

			// 购买
			auth.POST("/chapters/:id/purchase", handlers.PurchaseChapter)
			auth.POST("/content/:id/purchase", handlers.PurchaseContent)
			auth.POST("/membership/purchase", handlers.PurchaseMembership)
			auth.GET("/membership/upgrade-preview", handlers.GetMembershipUpgradePreview)

			// 新会员体系（1.0.5 双端启用）
			auth.GET("/membership-v2/me", handlers.GetMyMembershipV2)
			auth.POST("/membership-v2/join", handlers.JoinMembershipV2)
			auth.POST("/membership-v2/task", handlers.PostMembershipTask)
			auth.POST("/membership-v2/subscription", handlers.ApplyMembershipSubscription)
			auth.POST("/membership-v2/subscribe", handlers.SubscribeMembershipV2) // 下单（创建 KikiPay 订单，无自动续费）

			// 余额充值
			auth.POST("/balance/recharge", handlers.RechargeBalance)

			// 订单
			auth.GET("/orders", handlers.GetOrders)
			auth.GET("/orders/:orderNo/status", handlers.GetOrderStatus)
			auth.POST("/orders/:orderNo/complete", handlers.CompleteOrder)
			auth.POST("/orders/:orderNo/pay", handlers.CreatePaymentOrder)

			// 优惠券
			auth.POST("/coupons/validate", handlers.ValidateCoupon)

			// ===== Zibll: 论坛 =====
			auth.POST("/bbs/topics", handlers.CreateTopic)
			auth.PUT("/bbs/topics/:id", handlers.UpdateTopic)
			auth.DELETE("/bbs/topics/:id", handlers.DeleteTopic)
			auth.POST("/bbs/topics/:topicId/replies", handlers.CreateReply)
			auth.DELETE("/bbs/replies/:id", handlers.DeleteReply)
			auth.POST("/bbs/like", handlers.ToggleLike)

			// ===== Zibll: 评论 =====
			auth.POST("/articles/:id/comments", handlers.CreateComment)
			auth.POST("/comments/upload", handlers.UploadCommentImage)
			auth.POST("/articles/:id/report-dead-link", handlers.ReportDeadLink)
			auth.DELETE("/comments/:id", handlers.DeleteComment)

			// ===== Zibll: 收藏 =====
			auth.GET("/favorites", handlers.GetFavorites)
			auth.POST("/favorites/toggle", handlers.ToggleFavorite)
			auth.GET("/favorites/check", handlers.CheckFavorite)

			// ===== Zibll: 私信 =====
			auth.GET("/pm/conversations", handlers.GetConversationList)
			auth.GET("/pm/conversations/:userId", handlers.GetPrivateMessages)
			auth.POST("/pm/send", handlers.SendPrivateMessage)
			auth.GET("/pm/unread", handlers.GetUnreadPrivateCount)

			// ===== Zibll: 用户 ↔ 管理员 客服聊天 =====
			auth.GET("/chat/open", handlers.OpenChat)
			auth.POST("/chat/send", handlers.SendChatMessage)
			auth.POST("/chat/upload", handlers.UploadChatImage)
			auth.GET("/chat/messages", handlers.GetChatMessages)
			auth.GET("/chat/unread", handlers.GetChatUnread)
			auth.POST("/chat/end", handlers.EndChat)
			auth.POST("/chat/clear", handlers.ClearChat)
			auth.GET("/chat/history", handlers.GetChatHistory)

			// ===== 坐席侧客服（登录用户且为已注册坐席）=====
			auth.GET("/cs/conversations", handlers.CSAgentConversations)
			auth.GET("/cs/conversations/:id/messages", handlers.CSAgentMessages)
			auth.POST("/cs/conversations/:id/reply", handlers.CSAgentReply)
			auth.POST("/cs/agent/online", handlers.CSAgentSetOnline)
			auth.POST("/cs/agent/offline", handlers.CSAgentSetOffline)

			// ===== Zibll: 邀请 =====
			auth.GET("/invite/my-code", handlers.GetMyInviteCode)
			auth.GET("/invite/records", handlers.GetInviteRecords)

			// ===== Zibll: 打赏 =====
			auth.POST("/reward", handlers.CreateReward)

			// ===== Zibll: 积分商城 =====
			auth.POST("/points-exchange/:id", handlers.ExchangePointsProduct)
			auth.GET("/points-exchange/history", handlers.GetExchangeHistory)

			// ===== Zibll: VIP卡密 =====
			auth.POST("/vip-card/redeem", handlers.RedeemVipCard)

			// ===== Zibll: 任务 =====
			auth.GET("/tasks/daily", handlers.GetDailyTasks)
			auth.POST("/tasks/complete", handlers.CompleteTask)

			// ===== 积分概览 =====
			auth.GET("/points/mine", handlers.GetMyPoints)

			// ===== 文章付费下载 =====
			auth.POST("/articles/:id/attachments", handlers.CreateAttachment)
			auth.PUT("/attachments/:attachId", handlers.UpdateAttachment)
			auth.DELETE("/attachments/:attachId", handlers.DeleteAttachment)
			auth.POST("/attachments/:attachId/purchase", handlers.PurchaseAttachment)

			// 查看链接（记录查看行为）
			auth.POST("/attachments/:attachId/view-link", handlers.ViewAttachmentLink)
			// 用户附件购买记录
			auth.GET("/user/attachment-downloads", handlers.GetUserAttachmentDownloads)
			// 「我已拥有」的文章（按文章去重，仅返回仍在有效期内的解锁）——游戏档案用
			auth.GET("/user/purchased-articles", handlers.GetUserPurchasedArticles)
			// 会员每日免费下载额度（供前端简化付费卡片展示）
			auth.GET("/me/download-quota", handlers.GetDownloadQuota)

			// ===== 封面图本地化（登录即可）=====
			auth.POST("/localize-cover", handlers.LocalizeCover)

			// ===== 视频源管理 =====
			auth.POST("/articles/:id/video-sources", handlers.AddVideoSource)
			auth.PUT("/video-sources/:sourceId", handlers.UpdateVideoSource)
			auth.DELETE("/video-sources/:sourceId", handlers.DeleteVideoSource)
			auth.POST("/video-sources/:sourceId/fetch-cover", handlers.FetchVideoCover)
			auth.GET("/articles/:id/video-key", handlers.GetVideoDecryptKey)
			auth.POST("/video/progress", handlers.SaveVideoProgress)
			auth.GET("/video/progress", handlers.GetVideoProgress)
			// 文章/短剧点赞
			auth.POST("/articles/:id/like", handlers.ToggleArticleLike)
			// 分享积分上报（真实分享动作触发；share 任务为 server_only，不走 /tasks/complete）
			auth.POST("/tasks/share", handlers.ShareTask)

			// ===== 内容访问规则管理 =====
			auth.POST("/content-access-rules", handlers.SaveContentAccessRule)

			// ===== 红包雨（登录用户）=====
			auth.POST("/redpacket/claim", handlers.ClaimRedPacket)
		}

		// ===== Zibll: 公开接口（可选认证） =====
		api.GET("/bbs/forums", handlers.GetForums)
		api.GET("/bbs/topics", handlers.GetTopics)
		api.GET("/bbs/topics/:id", handlers.GetTopic)
		api.GET("/articles/:id/comments", handlers.GetComments)
		api.GET("/articles/:id/like", middleware.OptionalAuth(), handlers.GetArticleLike)
		api.GET("/bbs/replies", handlers.GetReplies)
		api.GET("/points-products", handlers.GetPointsProducts)
		api.GET("/points-products/:id", handlers.GetPointsProduct)
		api.GET("/rankings", handlers.GetRankings)
		api.GET("/search", handlers.Search)

		// ===== Zibll: 邀请码使用（公开） =====
		api.POST("/invite/use", handlers.UseInviteCode)

		// ===== 推广：后台上传的客户端安装包（公开） =====
		api.GET("/promo/installer", handlers.GetPromoInstaller)

		// ===== 文章附件（公开查看，可选认证） =====
		api.GET("/articles/:id/attachments", middleware.OptionalAuth(), handlers.GetArticleAttachments)
		api.GET("/attachments/:attachId/download", middleware.Auth(), handlers.DownloadAttachment)
		api.GET("/attachments/:attachId/check", middleware.Auth(), handlers.CheckAttachmentPurchase)

		// ===== 图片代理（公开）=====
		api.GET("/proxy-image", handlers.ProxyImage)

		// ===== GitHub 透明代理（公开）=====
		// 普通用户无法直连 GitHub 时，由后端解析并流式转发 api.github.com / objects.githubusercontent.com 等，
		// 实现客户端更新检查与安装包下载加速。仅白名单内 GitHub 域名可代理（防 SSRF），复用 github_token。
		api.GET("/client/gh-proxy", handlers.GithubProxy)

		// ===== P2P 图片服务（纯 P2P：服务器只索引+中转） =====
		api.GET("/p2p/img", handlers.ServeImageP2P)   // 高层取图（P2P优先，回退原图）
		api.GET("/p2p/locate", handlers.LocateImage)  // 查询图是否可被 P2P 取得
		api.GET("/p2p/fetch", handlers.FetchTransfer) // 取走中继字节（可选认证）
		// 需要登录的 P2P 写操作
		api.POST("/p2p/register", middleware.Auth(), handlers.RegisterImagePeer)
		api.GET("/p2p/pending", middleware.Auth(), handlers.PendingTransfers)
		api.POST("/p2p/deliver", middleware.Auth(), handlers.DeliverTransfer)

		// ===== P2P 资源（百度直链文件分块 + 视频/通用 blob 分片） =====
		api.GET("/p2p/resource/manifest", handlers.ResourceManifest)
		api.GET("/p2p/resource/chunk", handlers.ResourceChunk)
		api.POST("/p2p/resource/register", middleware.Auth(), handlers.RegisterResourcePeer)
		api.GET("/p2p/resource/locate", handlers.LocateResourceChunk)
		api.GET("/p2p/resource/pending", middleware.Auth(), handlers.PendingResourceChunks)
		api.POST("/p2p/resource/deliver", middleware.Auth(), handlers.DeliverResourceChunk)
		api.GET("/p2p/resource/fetch", handlers.FetchResourceChunk)
		api.POST("/p2p/blob/register", middleware.Auth(), handlers.RegisterBlobPeer)
		api.GET("/p2p/blob/locate", handlers.LocateBlob)
		api.GET("/p2p/blob/pending", middleware.Auth(), handlers.PendingBlobs)
		api.POST("/p2p/blob/deliver", middleware.Auth(), handlers.DeliverBlob)
		api.GET("/p2p/blob/fetch", handlers.FetchBlob)
		api.GET("/p2p/blob/img", handlers.ServeBlobP2P)

		// ===== 视频相关（公开） =====
		api.GET("/articles/:id/video-sources", handlers.GetVideoSources)
		api.GET("/video/parse-m3u8", handlers.ParseM3U8)
		api.GET("/video/proxy-m3u8", handlers.ProxyM3U8Playlist)
		api.GET("/video/proxy-ts", handlers.ProxyTSSegment)
		api.GET("/video/proxy-x", handlers.ProxyXVideo)
		api.GET("/video/parse-x", handlers.ParseXVideo)
		api.GET("/video/parse-x-info", handlers.ParseXVideoInfo)
		api.GET("/video/parse-info", handlers.ParseVideoInfo)
		api.GET("/video/cached/:articleId/:sourceId", handlers.ServeCachedVideo)
		api.POST("/video/recache/:articleId/:sourceId", handlers.ReCacheXVideo)
		api.POST("/video/extract-thumbnail", handlers.ExtractVideoThumbnail)

		// ===== 内容访问规则（公开查看） =====
		api.GET("/content-access-rules", handlers.GetContentAccessRule)
		api.GET("/content/check-access", handlers.CheckChapterAccess)

		// ===== 百度网盘解析（需登录） =====
		api.POST("/client/baidu/parse", middleware.Auth(), handlers.BaiduParseLink)
		api.POST("/client/baidu/direct-link", middleware.Auth(), handlers.BaiduGetDirectLink)
		api.POST("/client/baidu/check-cookie", middleware.Auth(), handlers.BaiduCheckCookie)

		// ===== 百度网盘扫码登录 & 用户 Cookie 绑定 =====
		api.GET("/client/baidu/qrcode", middleware.Auth(), handlers.BaiduGetQRCode)
		api.GET("/client/baidu/qrcode/check", middleware.Auth(), handlers.BaiduCheckQRCode)
		api.POST("/client/baidu/user-cookie", middleware.Auth(), handlers.BaiduSaveUserCookie)
		api.GET("/client/baidu/user-cookie", middleware.Auth(), handlers.BaiduGetUserCookie)

		// ===== 百度网盘代理下载 =====
		api.GET("/client/baidu/proxy-download", middleware.Auth(), handlers.BaiduProxyDownload)

		// ===== 视频加密密钥生成 =====
		api.GET("/video/generate-key", handlers.GenerateVideoKey)

		// ===== PocketBase 后台专用：会员图标上传 =====
		// 凭据 = PB 超管 token（gin 反向调 PB 的 auth-refresh 校验），
		// 故不能挂 Auth()/AdminRequired()（那两件套只认 gin 自己的用户 JWT）。
		api.POST("/pb-upload-membership-icon", handlers.PbUploadMembershipIcon)

		// 管理员接口
		admin := api.Group("/admin")
		admin.Use(middleware.Auth(), middleware.AdminRequired(), middleware.RecordActiveSource())
		{
			// 仪表盘
			admin.GET("/dashboard", handlers.GetDashboard)
			admin.GET("/dashboard/today", handlers.GetDashboardToday)

			// 文章管理
			admin.GET("/articles", handlers.GetArticlesAdmin)
			admin.POST("/articles", handlers.CreateArticle)
			admin.PUT("/articles/:id", handlers.UpdateArticle)
			admin.DELETE("/articles/:id", handlers.DeleteArticle)
			admin.POST("/articles/batch-delete", handlers.BatchDeleteArticles)
			admin.POST("/articles/batch-change-category", handlers.BatchChangeArticleCategory)
			admin.POST("/articles/batch-change-status", handlers.BatchChangeArticleStatus)
			admin.POST("/articles/batch-change-module", handlers.BatchChangeArticleModule)
			admin.POST("/articles/batch-set-price", handlers.BatchSetArticlePrice)
			admin.POST("/articles/batch-change-genre", handlers.BatchChangeArticleGenre)
			admin.POST("/articles/batch-auto-match-genre", handlers.BatchAutoMatchGenre)
			admin.GET("/articles/export", handlers.ExportArticlesTemplate)
			admin.GET("/articles/import-template", handlers.DownloadImportTemplate)
			admin.GET("/articles/pending", handlers.GetPendingArticles)
			// 后台编辑页详情：★ 单数 /article/:id，与复数 /articles 组隔离，
			// 避免将来新增 /articles/xxx 静态子节点与 :id 抢同一槽位（设计文档 §0.3）
			admin.GET("/article/:id", handlers.GetArticleDetailAdmin)
			admin.POST("/articles/:id/approve", handlers.ApproveArticle)

			// 内容管理（视频/漫画/小说）
			admin.GET("/content", handlers.GetContentAdmin)
			admin.POST("/content", handlers.CreateArticle)
			admin.PUT("/content/:id", handlers.UpdateArticle)
			admin.DELETE("/content/:id", handlers.DeleteArticle)

			// 章节管理
			admin.POST("/chapters", handlers.CreateChapter)
			admin.PUT("/chapters/:id", handlers.UpdateChapter)
			admin.DELETE("/chapters/:id", handlers.DeleteChapter)

			// SEO 管理
			admin.GET("/seo/settings", handlers.SEOSettings)
			admin.PUT("/seo/settings", handlers.UpdateSEOSettings)
			admin.POST("/seo/push", handlers.PushSitemap)
			admin.POST("/seo/pseudo-original", handlers.PseudoOriginalArticle)
			admin.POST("/seo/insert-keywords", handlers.InsertHiddenKeywords)
			admin.POST("/seo/clean-keywords", handlers.CleanHiddenKeywords)
			admin.POST("/seo/insert-keywords-all", handlers.InsertHiddenKeywordsAll)
			admin.POST("/seo/clean-keywords-all", handlers.CleanHiddenKeywordsAll)

			// ===== GEO 管理（生成式引擎优化产出物状态看板）=====
			admin.GET("/geo/status", handlers.GeoStatus)

			// 爬虫访问监控
			admin.GET("/crawler-logs", handlers.CrawlerLogs)
			admin.GET("/crawler-logs/summary", handlers.CrawlerLogsSummary)

			// 订单管理
			admin.GET("/orders", handlers.GetAllOrders)
			admin.POST("/orders/batch-delete", handlers.BatchDeleteOrders)

			// 用户管理
			admin.GET("/users", handlers.GetUsers)
			admin.POST("/users", handlers.AdminCreateUser)
			admin.PUT("/users/:id", handlers.UpdateUser)
			admin.DELETE("/users/:id", handlers.DeleteUser)
			admin.POST("/users/batch-delete", handlers.BatchDeleteUsers)
			// 后台查看指定用户的下载/解锁记录（含免费额度、付费下载）
			admin.GET("/users/:id/downloads", handlers.AdminGetUserDownloads)
			// 管理员手动解绑设备（豁免 72h 冷却，客服/封禁场景）
			admin.PUT("/users/:id/unbind-device", handlers.AdminUnbindDevice)

			// 优惠券管理
			admin.GET("/coupons", handlers.GetCoupons)
			admin.POST("/coupons", handlers.CreateCoupon)
			admin.PUT("/coupons/:id", handlers.UpdateCoupon)
			admin.DELETE("/coupons/:id", handlers.DeleteCoupon)
			admin.POST("/coupons/batch-delete", handlers.BatchDeleteCoupons)
			admin.POST("/coupons/generate", handlers.GenerateCoupons)
			admin.GET("/coupons/export", handlers.ExportCoupons)

			// 分类管理
			admin.GET("/categories", handlers.AdminGetCategories)
			admin.POST("/categories", handlers.AdminCreateCategory)
			admin.PUT("/categories/:id", handlers.AdminUpdateCategory)
			admin.DELETE("/categories/:id", handlers.AdminDeleteCategory)

			// 会员等级管理
			admin.GET("/membership-levels", handlers.AdminGetMembershipLevels)
			admin.POST("/membership-levels", handlers.AdminCreateMembershipLevel)
			admin.PUT("/membership-levels/:key", handlers.AdminUpdateMembershipLevel)
			admin.DELETE("/membership-levels/:key", handlers.AdminDeleteMembershipLevel)
			admin.POST("/membership-levels/seed", handlers.AdminSeedMembershipLevels)

			// 新会员体系（双组 VIP/SVIP + 经验等级，1.0.5 双端启用）
			admin.GET("/membership-v2", handlers.AdminListMembershipV2)
			admin.PUT("/membership-v2/groups/:groupKey", handlers.AdminUpdateMembershipGroup)
			admin.POST("/membership-v2/tiers", handlers.AdminCreateMembershipTier)
			admin.PUT("/membership-v2/tiers/:id", handlers.AdminUpdateMembershipTier)
			admin.DELETE("/membership-v2/tiers/:id", handlers.AdminDeleteMembershipTier)
			admin.GET("/membership-v2/task-exp", handlers.AdminGetMembershipTaskExp)
			admin.PUT("/membership-v2/task-exp", handlers.AdminUpdateMembershipTaskExp)
			admin.POST("/membership-v2/seed", handlers.AdminSeedMembershipV2)

			// 导航菜单（分类自动生成 + 自定义菜单）
			admin.GET("/navigations", handlers.AdminGetNavigations)
			admin.PUT("/navigations/batch-sort", handlers.AdminBatchUpdateNavigationSort)
			admin.PUT("/navigations/:id/toggle", handlers.AdminToggleNavStatus)
			admin.POST("/navigations/custom", handlers.AdminCreateCustomNav)
			admin.PUT("/navigations/custom/:id", handlers.AdminUpdateCustomNav)
			admin.DELETE("/navigations/custom/:id", handlers.AdminDeleteCustomNav)
			admin.POST("/navigations/sync-categories", handlers.AdminSyncCategoriesToNavigation)

			// ===== 单页（独立页面）管理 =====
			admin.GET("/pages", handlers.AdminGetPages)
			admin.POST("/pages", handlers.AdminCreatePage)
			admin.PUT("/pages/:id", handlers.AdminUpdatePage)
			admin.DELETE("/pages/:id", handlers.AdminDeletePage)

			// ===== 文章评论管理 =====
			admin.GET("/comments", handlers.AdminGetComments)
			admin.POST("/comments/batch-delete", handlers.AdminBatchDeleteComments)
			admin.PUT("/comments/:id/status", handlers.AdminUpdateCommentStatus)
			admin.DELETE("/comments/:id", handlers.AdminDeleteComment)

			// ===== 标签管理 =====
			admin.GET("/tags", handlers.AdminGetTags)
			admin.POST("/tags/rename", handlers.AdminRenameTag)
			admin.POST("/tags/batch-delete", handlers.AdminBatchDeleteTags)

			// 工具：修复无缩略图文章
			admin.POST("/fix-thumbnails", handlers.AdminFixThumbnails)

			// Excel导入文章
			admin.POST("/articles/import-excel", handlers.ImportArticlesFromExcel)

			// 定时导入文章（固定/随机每天 N 篇）
			admin.POST("/articles/import-scheduled", handlers.ImportArticlesScheduled)
			admin.GET("/scheduled/list", handlers.ScheduledPublishList)
			admin.POST("/scheduled/cancel", handlers.ScheduledPublishCancel)

			// 工具：数据库冗余清理 / 旧文件缓存清理
			admin.POST("/tools/db-cleanup", handlers.DBCleanup)
			admin.POST("/tools/files-cleanup", handlers.FilesCleanup)

			// 配置管理
			admin.GET("/settings", handlers.GetAllSettingsHandler)
			admin.PUT("/settings", handlers.UpdateSettings)
			admin.PUT("/settings/site", handlers.UpdateSiteSettings)
			admin.PUT("/settings/membership", handlers.UpdateMembershipConfig)
			admin.PUT("/settings/payment", handlers.UpdatePaymentConfig)
			admin.PUT("/settings/content", handlers.UpdateContentConfig)
			admin.PUT("/settings/register", handlers.UpdateRegisterConfig)
			admin.PUT("/settings/email", handlers.UpdateEmailConfig)
			admin.PUT("/settings/content-nav", handlers.UpdateContentNavConfig)
			admin.PUT("/settings/app-update", handlers.UpdateAppUpdateConfig)
			admin.PUT("/settings/storage", handlers.UpdateStorageConfig)
			// ===== 邮件群发 =====
			admin.GET("/email/status", handlers.GetEmailCampaignStatus)
			admin.POST("/email/import", handlers.ImportEmailRecipients)
			admin.POST("/email/send-batch", handlers.SendEmailBatch)
			admin.POST("/email/reset", handlers.ResetEmailCampaign)

			// ===== 站内通知群发 =====
			admin.GET("/notifications", handlers.AdminGetNotifications)
			admin.GET("/notifications/targets", handlers.AdminNotificationTargets)
			admin.POST("/notifications", handlers.AdminSendNotification)
			admin.DELETE("/notifications/:id", handlers.AdminDeleteNotification)
			admin.POST("/notifications/:id/recall", handlers.AdminRecallNotification)

			// R2 存量文件迁移
			admin.POST("/r2/migrate", handlers.StartR2Migration)
			admin.GET("/r2/migrate/status", handlers.GetR2MigrationStatus)

			// 系统管理
			admin.POST("/system/reinstall", handlers.Reinstall)
			admin.POST("/system/clear", handlers.ClearAllData)
			admin.GET("/export", handlers.ExportData)

			// ===== Zibll 管理: 论坛 =====
			admin.POST("/bbs/forums", handlers.CreateForum)
			admin.PUT("/bbs/forums/:id", handlers.UpdateForum)
			admin.DELETE("/bbs/forums/:id", handlers.DeleteForum)
			admin.POST("/bbs/topics/:id/action", handlers.AdminTopicAction)

			// ===== Zibll 管理: 积分商城 =====
			admin.POST("/points-products", handlers.AdminCreatePointsProduct)
			admin.PUT("/points-products/:id", handlers.AdminUpdatePointsProduct)
			admin.DELETE("/points-products/:id", handlers.AdminDeletePointsProduct)

			// ===== 积分配置 / 调分 =====
			admin.GET("/points/config", handlers.AdminGetPointsConfig)
			admin.PUT("/points/config", handlers.AdminSetPointsConfig)
			admin.POST("/points/adjust", handlers.AdminAdjustUserPoints)
			// 行为积分策略（每日任务发分配置）
			admin.GET("/points/tasks", handlers.AdminListDailyTasks)
			admin.PUT("/points/tasks/:id", handlers.AdminUpdateDailyTask)

			// ===== Zibll 管理: 任务 =====
			admin.GET("/tasks", handlers.AdminGetTasks)
			admin.POST("/tasks", handlers.AdminCreateTask)
			admin.PUT("/tasks/:id", handlers.AdminUpdateTask)
			admin.DELETE("/tasks/:id", handlers.AdminDeleteTask)

			// ===== Zibll 管理: VIP卡密 =====
			admin.POST("/vip-cards/generate", handlers.GenerateVipCards)
			admin.GET("/vip-cards", handlers.GetVipCards)
			admin.DELETE("/vip-cards/:id", handlers.DeleteVipCard)

			// ===== Zibll 管理: 积分商品（独立管理接口） =====
			admin.POST("/points-products/manage", handlers.AdminCreatePointsProduct)
			admin.PUT("/points-products/manage/:id", handlers.AdminUpdatePointsProduct)
			admin.DELETE("/points-products/manage/:id", handlers.AdminDeletePointsProduct)

			// ===== Zibll 管理: 漫画/视频访问规则 =====
			admin.GET("/content-access-rules/all", handlers.AdminGetAllAccessRules)

			// ===== Banner 管理 =====
			admin.GET("/banners", handlers.AdminGetBanners)
			admin.POST("/banners", handlers.AdminCreateBanner)
			admin.PUT("/banners/:id", handlers.AdminUpdateBanner)
			admin.DELETE("/banners/:id", handlers.AdminDeleteBanner)

			// ===== 首页模块管理 =====
			admin.GET("/home-modules", handlers.AdminGetHomeModules)
			admin.POST("/home-modules", handlers.AdminCreateHomeModule)
			admin.PUT("/home-modules/:id", handlers.AdminUpdateHomeModule)
			admin.DELETE("/home-modules/:id", handlers.AdminDeleteHomeModule)

			// ===== 客户端配置管理（登录背景 / 客户端菜单） =====
			admin.GET("/client-config", handlers.AdminGetClientConfig)
			admin.PUT("/client-config", handlers.AdminUpdateClientConfig)
			// ===== 安卓端配置管理（接口域名 / 分类ID / 游戏类型 / 强制更新等）=====
			admin.GET("/android-config", handlers.AdminGetAndroidConfig)
			admin.PUT("/android-config", handlers.AdminUpdateAndroidConfig)
			// 后台代拉 GitHub 最新 Release（验证配置 + 回填版本号）
			admin.GET("/github-release/latest", handlers.AdminGithubLatestRelease)
			// 后台把安装包发布到 GitHub Releases（复用 github_token）
			admin.POST("/github-release/upload", handlers.AdminGithubUploadRelease)
			// PCDN 节点观测
			admin.GET("/pcdn/stats", handlers.PcdnStats)
			admin.POST("/redpacket", handlers.AdminUpdateRedPacket)

			// ===== PocketBase 配置域同步（PB 是配置唯一写方，gin 定时拉取；此接口手动立即触发一轮）=====
			admin.POST("/pb-config-sync", handlers.AdminTriggerPbSync)

			// ===== 首页 Banner 管理（电脑首页 / 手机首页）=====
			admin.POST("/upload-banner", handlers.UploadBanner)
			// ===== 站点 Logo 上传（支持 SVG）=====
			admin.POST("/upload-logo", handlers.UploadLogo)
			// ===== 封面 / 通用 / 安装包上传（仅管理员）=====
			admin.POST("/upload-cover", handlers.UploadCover)
			admin.POST("/upload-generic", handlers.UploadGeneric)
			admin.POST("/upload-installer", handlers.UploadInstaller)
			// ===== 会员体系图标上传（组图标 / 等级图标：ico/png/svg 等）=====
			admin.POST("/upload-membership-icon", handlers.UploadMembershipIcon)
			admin.PUT("/settings/home-banner", handlers.UpdateHomeBannerConfig)

			// ===== 首页分类目录管理（电脑首页 / 手机首页，拖拽排序）=====
			admin.PUT("/settings/home-categories", handlers.UpdateHomeCategoriesConfig)

			// ===== 首页展示配置管理（卡片风格 / 每行几个，按设备）=====
			admin.PUT("/settings/home-display", handlers.UpdateHomeDisplayConfig)
			admin.GET("/settings/home-display", handlers.GetHomeDisplayConfig)

			// ===== 首页 Banner 读取（管理端，按设备）=====
			admin.GET("/settings/home-banner", handlers.GetHomeBanner)

			// ===== 重新同步下载资源的游戏分类 =====
			admin.POST("/articles/resync-game-fields", handlers.AdminResyncGameFields)

			// ===== 重新生成文章简介 =====
			admin.POST("/articles/resync-summaries", handlers.AdminResyncSummaries)

			// ===== 百度网盘 Cookie 管理 =====
			admin.GET("/baidu-cookie", handlers.AdminGetBaiduCookie)
			admin.GET("/baidu-cookie/status", handlers.AdminBaiduCookieStatus)
			admin.PUT("/baidu-cookie", handlers.AdminUpdateBaiduCookie)
			admin.POST("/baidu-cookie/test", handlers.AdminTestBaiduCookie)
			admin.POST("/baidu/share", handlers.AdminBaiduCreateShare)
			admin.POST("/baidu/migrate", handlers.AdminBaiduMigrate)
			admin.POST("/baidu/resolve", handlers.AdminBaiduResolve)
			admin.GET("/baidu/migrations", handlers.AdminBaiduMigrations)

			// ===== X.com 视频解析测试 =====
			admin.GET("/x-video-test", handlers.TestXVideoParse)

			// ===== DeepSeek AI 自动新闻 =====
			admin.POST("/ai/generate-news", handlers.GenerateNewsNow)
			admin.GET("/ai/news-status", handlers.GetNewsStatus)

			// ===== API 资源采集（MacCMS XML 源） =====
			admin.POST("/collect/run", handlers.CollectRun)
			admin.GET("/collect/status", handlers.CollectStatus)
			admin.GET("/collect/config", handlers.CollectConfigGet)
			admin.POST("/collect/config", handlers.CollectConfigSave)
			admin.POST("/collect/clear", handlers.CollectClear)
			admin.GET("/collect/source-categories", handlers.CollectSourceCategories)

			// ===== 用户 ↔ 管理员 客服聊天 =====
			admin.GET("/chat/conversations", handlers.AdminGetChatConversations)
			admin.GET("/chat/messages", handlers.AdminGetChatMessages)
			admin.POST("/chat/reply", handlers.AdminReplyChat)
			// 客服自动回复规则管理
			admin.GET("/chat/auto-replies", handlers.AdminListAutoReplies)
			admin.POST("/chat/auto-replies", handlers.AdminCreateAutoReply)
			admin.PUT("/chat/auto-replies/:id", handlers.AdminUpdateAutoReply)
			admin.DELETE("/chat/auto-replies/:id", handlers.AdminDeleteAutoReply)
			// 客服开场问候语
			admin.GET("/chat/greeting", handlers.AdminGetChatGreeting)
			admin.PUT("/chat/greeting", handlers.AdminUpdateChatGreeting)

			// ===== 多端统一配置（参数 / 功能开关 / 策略） =====
			admin.GET("/multi-config/:clientType", handlers.AdminGetMergedConfig)
			admin.PUT("/multi-config/:clientType", handlers.AdminUpdateClientParam)
			admin.GET("/feature-flags", handlers.AdminListFeatureFlags)
			admin.POST("/feature-flags", handlers.AdminCreateFeatureFlag)
			admin.PUT("/feature-flags/:id", handlers.AdminUpdateFeatureFlag)
			admin.DELETE("/feature-flags/:id", handlers.AdminDeleteFeatureFlag)
			admin.GET("/strategies", handlers.AdminListStrategies)
			admin.POST("/strategies", handlers.AdminCreateStrategy)
			admin.PUT("/strategies/:id", handlers.AdminUpdateStrategy)
			admin.DELETE("/strategies/:id", handlers.AdminDeleteStrategy)
			// 网站端配置（消除孤儿 web-config）
			admin.GET("/web-config", handlers.AdminGetWebConfig)
			admin.PUT("/web-config", handlers.AdminUpdateWebConfig)

			// ===== 在线客服：坐席 / 分配 / 接管 / 转接 / 监控 =====
			admin.GET("/cs/agents", handlers.CSListAgents)
			admin.POST("/cs/agents", handlers.CSCreateAgent)
			admin.PUT("/cs/agents/:id", handlers.CSUpdateAgent)
			admin.DELETE("/cs/agents/:id", handlers.CSDeleteAgent)
			admin.POST("/cs/conversations/:id/assign", handlers.CSAssign)
			admin.POST("/cs/conversations/:id/takeover", handlers.CSTakeover)
			admin.POST("/cs/conversations/:id/transfer", handlers.CSTransfer)
			admin.POST("/cs/conversations/:id/close", handlers.CSCloseConv)
			admin.GET("/cs/queue", handlers.CSQueue)
			admin.GET("/cs/monitor", handlers.CSMonitor)
			admin.GET("/cs/routing-rules", handlers.CSListRoutingRules)
			admin.POST("/cs/routing-rules", handlers.CSCreateRoutingRule)
			admin.PUT("/cs/routing-rules/:id", handlers.CSUpdateRoutingRule)
			admin.DELETE("/cs/routing-rules/:id", handlers.CSDeleteRoutingRule)
			admin.PUT("/cs/routing", handlers.CSUpdateRouting)
		admin.GET("/cs/routing", handlers.CSGetRouting)

			// ===== 签到计划配置 =====
			admin.GET("/checkin-plan", handlers.AdminGetCheckinPlan)
			admin.PUT("/checkin-plan", handlers.AdminUpdateCheckinPlan)
		}

		// 支付回调（无需认证）
		api.GET("/payment/kikipay/config", handlers.GetKikiPayConfig)
		api.POST("/payment/kikipay/order", handlers.CreateKikiPayOrder)
		api.GET("/payment/kikipay/query", handlers.QueryKikiPayOrder)
		api.POST("/payment/kikipay/notify", handlers.KikiPayNotify)
		api.GET("/payment/kikipay/notify", handlers.KikiPayNotify)
		api.POST("/payment/notify", handlers.PaymentNotify)
		api.GET("/payment/notify", handlers.PaymentNotify)
	}

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("🚀 TechLife Blog 后端服务启动: http://localhost:%s\n", port)
	// 启动 DeepSeek AI 每日新闻调度器（随服务进程运行）
	handlers.StartNewsScheduler()
	// 启动 API 资源采集每日调度器
	handlers.StartCollectorScheduler()
	// 启动邮件群发每日调度器
	handlers.StartEmailCampaignScheduler()
	// 启动定时发布调度器（到期自动翻转为已发布）
	handlers.StartScheduledPublishScheduler()
	// 启动 KikiPay 服务端对账调度器（周期性补单，根治「付了钱没会员」）
	handlers.StartKikiPayReconcileScheduler()
	// 启动 PocketBase 配置域同步（PB 改配置 → gin 定时拉取落库；PB 唯一写方）
	handlers.StartPbConfigSyncScheduler()
	// 限流器后台清理（每分钟回收过期限流桶，避免内存只增不减）
	middleware.StartRateLimitSweeper()

	fmt.Printf("📦 API 地址: http://localhost:%s/api\n", port)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
