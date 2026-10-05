package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"techlife-blog/database"
	"techlife-blog/models"
	"techlife-blog/utils"

	"github.com/gin-gonic/gin"
)

// GetArticles 获取文章列表
func GetArticles(c *gin.Context) {
	contentType := c.Query("contentType")
	tag := c.Query("tag")
	search := c.Query("search")
	if search == "" {
		search = c.Query("q") // 前端搜索统一传 q
	}
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("pageSize"), 6)
	clientType := c.Query("clientType") // desktop / mobile

	query := database.DB.Model(&models.Article{}).Where("status = ?", "published")

	// 按分类筛选（分类即菜单，categoryId 优先）。
	// 当使用 categoryId / menuId 时，不再叠加 content_type 过滤，避免分类页因为默认 article 而空白。
	menuId := c.Query("menuId")
	categoryId := c.Query("categoryId")
	if categoryId != "" {
		query = query.Where("category_id = ?", categoryId)
	} else if menuId != "" {
		// 兼容旧数据：menu_id 可能指向旧的 NavigationMenu ID
		query = query.Where("(menu_id = ? OR (menu_id IS NULL AND category_id = ?))", menuId, menuId)
	}

	// 按内容类型筛选：仅在没有指定分类/菜单时才生效；未指定时默认 article，保持首页等未传参接口行为
	if categoryId == "" && menuId == "" {
		if contentType == "" {
			contentType = "article"
		}
		if contentType != "" {
			query = query.Where("content_type = ?", contentType)
		}
	}

	// 按标签筛选
	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	// 搜索
	if search != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 客户端类型筛选（双端内容分发）
	if clientType == "mobile" {
		query = query.Where("attrs = '' OR attrs IS NULL OR attrs LIKE '%mobile_app%'")
	}

	// 游戏平台筛选（精确匹配：pc / mobile / both）
	gamePlatform := c.Query("gamePlatform")
	if gamePlatform != "" {
		query = query.Where("game_platform = ?", gamePlatform)
	}

	// 设备兼容性筛选（包容匹配，供原生 App 使用）
	// deviceClass=mobile -> 手机可玩（mobile + both），隐藏纯 PC 内容
	// deviceClass=pc     -> 电脑可玩（pc + both）
	switch c.Query("deviceClass") {
	case "mobile":
		query = query.Where("game_platform IN ?", []string{"mobile", "both"})
	case "pc":
		query = query.Where("game_platform IN ?", []string{"pc", "both"})
	}

	// 移动端游戏分类（games categoryId=3）：仅展示标题含「安卓」的资源。
	// 大量 PC 标记(attrs='xt_pc'/'xt_pc,xt_az')游戏无安卓版本，不在手机端出现；
	// 标题带「安卓」即视为安卓可玩版本（与 android-config.gamesCategoryId 保持一致，默认 3）。
	if c.Query("deviceClass") == "mobile" && categoryId == "3" {
		query = query.Where("title LIKE ?", "%安卓%")
	}

	// 游戏类型筛选（“其他”表示未设置类型的资源）
	gameGenre := c.Query("gameGenre")
	if gameGenre != "" {
		if gameGenre == "其他" {
			query = query.Where("game_genre = '' OR game_genre IS NULL")
		} else {
			query = query.Where("game_genre LIKE ?", "%"+gameGenre+"%")
		}
	}

	var total int64
	query.Count(&total)

	// 排序：latest（默认，按发布时间倒序）/ hot（按浏览量倒序）/ price（按价格升序）。
	// 末尾统一加 id desc 兜底：发布时间相同的文章（批量导入/回填常出现）顺序确定，
	// 与后台 admin 列表（admin.go 同样用 id desc 兜底）保持一致，避免客户端顺序与后台对不上。
	var orderClause string
	switch c.Query("sort") {
	case "hot":
		orderClause = "views desc, id desc"
	case "price":
		orderClause = "price asc, id desc"
	default:
		orderClause = "COALESCE(published_at, created_at) desc, id desc"
	}

	var articles []models.Article
	query.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles)

	// 列表/搜索接口处理：
	// 1) 若 cover 为空，从正文首图提取填入，保证前端列表缩略图正常显示；
	// 2) 清空正文 Content（列表前端只用 cover/标题，且避免付费文章正文里的网盘链接在列表 API 泄露）；
	// 3) 注入分类 slug / 名称，供前端统一输出 /[slug]/id.html 链接。
	var categories []models.Category
	database.DB.Select("id, slug, name, status").Where("status = ?", "active").Find(&categories)
	catMap := make(map[uint]models.Category, len(categories))
	for _, c := range categories {
		catMap[c.ID] = c
	}
	for i := range articles {
		if articles[i].Cover == "" && articles[i].Content != "" {
			articles[i].Cover = extractFirstImage(articles[i].Content)
		}
		articles[i].Content = ""
		if articles[i].CategoryID != nil && *articles[i].CategoryID > 0 {
			if cat, ok := catMap[*articles[i].CategoryID]; ok && cat.Slug != "" {
				articles[i].CategorySlug = cat.Slug
				articles[i].CategoryName = cat.Name
			}
		}
		if articles[i].CategorySlug == "" {
			articles[i].CategorySlug = "articles"
		}
	}

	// 批量注入视频集数：对结果集中的文章，按 article_id 分组统计 video_sources 数量（避免 N+1）。
	if len(articles) > 0 {
		ids := make([]uint, 0, len(articles))
		for _, a := range articles {
			ids = append(ids, a.ID)
		}
		type epCount struct {
			ArticleID uint `json:"articleId"`
			Cnt       int  `json:"cnt"`
		}
		var counts []epCount
		database.DB.Model(&models.VideoSource{}).
			Select("article_id, COUNT(*) as cnt").
			Where("article_id IN ?", ids).
			Group("article_id").
			Scan(&counts)
		epMap := make(map[uint]int, len(counts))
		for _, c := range counts {
			epMap[c.ArticleID] = c.Cnt
		}
		for i := range articles {
			// 优先使用文章自身录入的 episode_total（真实列，允许人工覆盖）；
			// 为 0（未填 / 存量数据）时回退到 video_sources 计数，与改造前行为一致。
			if articles[i].EpisodeTotal > 0 {
				articles[i].EpisodeCount = articles[i].EpisodeTotal
			} else {
				articles[i].EpisodeCount = epMap[articles[i].ID]
			}
		}
	}

	utils.Success(c, gin.H{
		"list":      articles,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"totalPage": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetArticle 获取文章详情
// articleDetailPayload 文章详情中与用户态无关的部分，用于预缓存，避免每次请求重复查库。
type articleDetailPayload struct {
	Article  models.Article
	Chapters []models.Chapter
}

// loadArticleDetail 加载文章正文 + 分类 + 章节（不含任何按用户态变化的内容）。
// 访问权限、浏览量、积分等按用户态逻辑在 GetArticle 中每次重新计算。
func loadArticleDetail(id string) (articleDetailPayload, error) {
	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		return articleDetailPayload{}, err
	}
	if article.CategoryID != nil && *article.CategoryID > 0 {
		var cat models.Category
		if database.DB.First(&cat, *article.CategoryID).Error == nil && cat.Slug != "" && cat.Status == "active" {
			article.CategorySlug = cat.Slug
			article.CategoryName = cat.Name
		}
	}
	if article.CategorySlug == "" {
		article.CategorySlug = "articles"
	}
	var chapters []models.Chapter
	if article.ContentType == "comic" || article.ContentType == "novel" {
		database.DB.Where("content_id = ?", article.ID).Order("chapter_order asc").Find(&chapters)
	}
	return articleDetailPayload{Article: article, Chapters: chapters}, nil
}

// WarmArticleCache 启动预热：把最近 n 篇已发布文章预载入缓存，使首发/爬虫流量首个请求即命中。
func WarmArticleCache(n int) {
	ids := []uint{}
	database.DB.Model(&models.Article{}).Where("status = ?", "published").Order("id desc").Limit(n).Pluck("id", &ids)
	count := 0
	for _, id := range ids {
		key := "detail:" + strconv.FormatUint(uint64(id), 10)
		if _, ok := utils.ArticleCache.Get(key); ok {
			continue
		}
		p, err := loadArticleDetail(strconv.FormatUint(uint64(id), 10))
		if err != nil {
			continue
		}
		utils.ArticleCache.Set(key, p)
		count++
	}
	log.Printf("[cache] 文章预缓存完成：预热 %d 篇（上限 %d）", count, n)
}

func GetArticle(c *gin.Context) {
	id := c.Param("id")
	cacheKey := "detail:" + id

	var p articleDetailPayload
	if cached, ok := utils.ArticleCache.Get(cacheKey); ok {
		p = cached.(articleDetailPayload)
	} else {
		loaded, err := loadArticleDetail(id)
		if err != nil {
			utils.Error(c, 404, "文章不存在")
			return
		}
		p = loaded
		utils.ArticleCache.Set(cacheKey, p)
	}
	article := p.Article
	chapters := p.Chapters

	// 增加浏览量（原子自增，避免命中缓存旧值后写回导致计数丢失）
	database.DB.Exec("UPDATE articles SET views = views + 1 WHERE id = ?", article.ID)

	// 注入分类 slug / 名称（给前端生成规范 URL /[slug]/id.html）
	if article.CategoryID != nil && *article.CategoryID > 0 {
		var cat models.Category
		if database.DB.First(&cat, *article.CategoryID).Error == nil && cat.Slug != "" && cat.Status == "active" {
			article.CategorySlug = cat.Slug
			article.CategoryName = cat.Name
		}
	}
	if article.CategorySlug == "" {
		article.CategorySlug = "articles"
	}

	// 章节直接复用预缓存中的 p.Chapters（漫画/小说类），命中缓存即免去一次查库；
	// 文章编辑后 UpdateArticle/PublishArticle 已失效该缓存键，至多 2 分钟 TTL 后自动刷新。

	// 检查用户是否有权访问
	uid := utils.GetUserID(c)
	hasAccess := true
	if article.MembersOnly {
		// ★ 仅会员可访问/下载：免费用户即使付费也不能访问，只有持有有效会员的用户（及管理员/作者）可访问
		hasAccess = false
		if uid > 0 {
			var user models.User
			database.DB.First(&user, uid)
			// 管理员直接放行
			if user.Role == "admin" {
				hasAccess = true
			}
			// 作者本人
			if !hasAccess && article.AuthorID == uid {
				hasAccess = true
			}
			// 有效会员放行
			if !hasAccess && user.MembershipLevel != "free" && user.MembershipLevel != "" {
				if user.MembershipExpiry == nil || user.MembershipExpiry.After(time.Now()) {
					hasAccess = true
				}
			}
		}
	} else if article.IsPremium && article.Price > 0 {
		hasAccess = false
		if uid > 0 {
			// 管理员直接放行
			var user models.User
			database.DB.First(&user, uid)
			if user.Role == "admin" {
				hasAccess = true
			}
			// 检查是否已购买
			if !hasAccess {
				var purchase models.Purchase
				if database.DB.Where("user_id = ? AND type = ? AND item_id = ?", uid, "content", article.ID).First(&purchase).Error == nil {
					hasAccess = true
				}
			}
			// 检查会员等级
			if !hasAccess && user.MembershipLevel != "free" && user.MembershipLevel != "" {
				// 检查会员是否过期
				if user.MembershipExpiry == nil || user.MembershipExpiry.After(time.Now()) {
					// ★ 检查文章VIP分级定价中该等级是否免费
					if article.VipPrices != "" {
						// 使用统一的 calculateArticlePrice 函数（含等级未匹配时的回退逻辑）
						price := calculateArticlePrice(article.Price, article.VipPrices, user.MembershipLevel)
						if price == 0 {
							hasAccess = true
						}
					} else {
						// 没有 vipPrices，付费会员默认免费
						hasAccess = true
					}
				}
			}
			// 作者本人
			if !hasAccess && article.AuthorID == uid {
				hasAccess = true
			}
		}
	}

	// 正文始终返回（对用户可见）。真正可下载的资源（网盘链接/提取码/本地文件）在
	// 附件接口 GetArticleAttachments 中按购买/会员状态保护，未购买不下发链接，
	// 所以正文可见不会泄露可用的付费资源。
	locked := !hasAccess && (article.MembersOnly || (article.IsPremium && article.Price > 0))

	// ★ 阅读积分：登录用户真实打开文章详情时发放 read_article 任务积分。
	//   - 匿名请求（uid == 0，如 SSR 预渲染、抓取器）不发，只有登录用户才计；
	//   - 幂等由 daily_tasks.max_daily（read_article = 1 次/天）+ UserTask 保证，
	//     同一天反复刷新同一篇文章也只发一次（awardPointsForAction 内部判重）；
	//   - 结果不回传给前端，前端任务列表靠重新拉取 /tasks/daily 展示完成状态。
	//   对应 db.go seedDailyTasks 中 read_article 的 ServerOnly=true。
	if uid > 0 {
		awardPointsForAction(uid, "read_article")
	}

	// 正文完整返回，封面/正文首图均保留显示（用户要求正文第一张图正常展示）
	utils.Success(c, gin.H{
		"article":   article,
		"chapters":  chapters,
		"hasAccess": hasAccess,
		"locked":    locked,
	})
}

// CreateArticle 创建文章
func CreateArticle(c *gin.Context) {
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		utils.Error400(c, "参数错误")
		return
	}

	uid := utils.GetUserID(c)
	username := utils.GetUsername(c)
	article.AuthorID = uid
	article.AuthorName = username

	// 普通用户发布的文章需要审核
	if !utils.IsAdmin(c) {
		article.Status = "pending"
	}

	// 如果没有缩略图，自动取正文第一张图片
	if article.Cover == "" && article.Content != "" {
		article.Cover = extractFirstImage(article.Content)
	}

	// 如果没有简介，自动从正文提取摘要
	if article.Summary == "" && article.Content != "" {
		article.Summary = generateSummaryFromContent(article.Content, 200)
	}

	// 自动检测游戏平台和类型（资源下载类型）
	autoDetectGameFields(&article)

	// 发布时间：仅当状态为「已发布」时记录实际发布时间（定时发布由调度器在翻转时写入）
	if article.Status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	}

	// ★ allow_comment 带 gorm:"default:true"，GORM 会把零值（false）从 INSERT 语句中剔除
	// 并回填 DB 默认值 → 前端「关闭评论」会被静默改成 true。
	// default:true 这个 tag 必须保留（AutoMigrate 给存量表加列时靠它把老文章填成 1），
	// 所以只在这里修 Create 路径：Create 【之前】捕获前端意图值，Create 【之后】强制写回。
	wantAllowComment := article.AllowComment

	if err := database.DB.Create(&article).Error; err != nil {
		utils.Error500(c, "创建失败")
		return
	}

	// 仅在需要纠正（用户明确要关闭评论）时补一条 UPDATE，正常情况不发多余写、也不动 updated_at。
	if !wantAllowComment {
		if err := database.DB.Model(&article).Update("allow_comment", wantAllowComment).Error; err != nil {
			log.Printf("[WARN] CreateArticle 回写 allow_comment=false 失败 id=%d: %v", article.ID, err)
		}
	}

	// 完成任务：发布文章（由真实发布动作触发，前端不可手动领取，防止刷分）
	completeTask(uid, "publish_article")

	// 自动同步 x.com 视频（补标题 + 创建视频源）
	go syncXVideoForArticle(article.ID)

	database.BumpContentVersion()
	utils.Success(c, article)
}

// UpdateArticle 更新文章
func UpdateArticle(c *gin.Context) {
	id := c.Param("id")
	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		utils.Error(c, 404, "文章不存在")
		return
	}

	var rawUpdates map[string]interface{}
	if err := c.ShouldBindJSON(&rawUpdates); err != nil {
		utils.Error400(c, "参数错误")
		return
	}

	// 按白名单过滤：丢弃未知 key（不报错，保证现有前端多传字段也不会 500）。
	// 不过滤的话，一个拼错/多传的 key 会让整条 UPDATE 报 "no such column"，
	// 同批次好字段全部写不进去，而旧代码不接 error → 前端拿到「保存成功」的假象。
	updates, dropped := filterArticleUpdates(rawUpdates)
	if len(dropped) > 0 {
		log.Printf("[WARN] UpdateArticle 丢弃未知字段 id=%s keys=%v", id, dropped)
	}

	oldStatus := article.Status
	if len(updates) > 0 {
		if err := database.DB.Model(&article).Updates(updates).Error; err != nil {
			utils.Error500(c, "更新失败: "+err.Error())
			return
		}
	}
	// 文章已变更，失效预缓存（TTL 内避免返回旧正文）
	utils.ArticleCache.Delete("detail:" + id)
	database.DB.First(&article, id)

	// 如果没有缩略图，自动取正文第一张图片
	if article.Cover == "" && article.Content != "" {
		cover := extractFirstImage(article.Content)
		if cover != "" {
			database.DB.Model(&article).Update("cover", cover)
		}
	}

	// 如果没有简介，自动从正文提取摘要
	if article.Summary == "" && article.Content != "" {
		summary := generateSummaryFromContent(article.Content, 200)
		if summary != "" {
			database.DB.Model(&article).Update("summary", summary)
		}
	}

	// 自动检测游戏平台和类型（资源下载类型，如果未手动设置）
	if article.ContentType == "download" {
		needUpdate := false
		if article.GamePlatform == "" {
			article.GamePlatform = detectGamePlatform(article.Title)
			needUpdate = true
		}
		if article.GameGenre == "" {
			article.GameGenre = detectGameGenre(article.Title)
			needUpdate = true
		}
		if needUpdate {
			database.DB.Model(&article).Updates(map[string]interface{}{
				"game_platform": article.GamePlatform,
				"game_genre":    article.GameGenre,
			})
		}
	}

	// 状态翻转为「已发布」时记录实际发布时间（用于「发布日期」展示）。
	// 仅当本次更新把 status 改为 published 且此前不是 published 才写入；已发布文章的普通编辑不受影响。
	if newStatus, ok := updates["status"].(string); ok && newStatus == "published" && oldStatus != "published" {
		now := time.Now()
		article.PublishedAt = &now
		database.DB.Model(&article).Update("published_at", now)
	}

	// 自动同步 x.com 视频（补标题 + 创建视频源）
	go syncXVideoForArticle(article.ID)

	database.BumpContentVersion()
	utils.Success(c, article)
}

// DeleteArticle 删除文章
func DeleteArticle(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Article{}, id).Error; err != nil {
		utils.Error500(c, "删除失败")
		return
	}
	// 同时删除章节
	database.DB.Where("content_id = ?", id).Delete(&models.Chapter{})
	database.BumpContentVersion()
	utils.SuccessMsg(c, "删除成功")
}

// GetTags 获取所有标签
func GetTags(c *gin.Context) {
	var articles []models.Article
	database.DB.Where("status = ?", "published").Select("tags").Find(&articles)

	tagCount := make(map[string]int)
	for _, a := range articles {
		if a.Tags == "" {
			continue
		}
		for _, tag := range strings.Split(a.Tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagCount[tag]++
			}
		}
	}

	type TagItem struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var tags []TagItem
	for name, count := range tagCount {
		tags = append(tags, TagItem{Name: name, Count: count})
	}

	utils.Success(c, tags)
}

// GetDashboard 获取仪表盘数据
func GetDashboard(c *gin.Context) {
	var articleCount, userCount, orderCount int64
	database.DB.Model(&models.Article{}).Count(&articleCount)
	database.DB.Model(&models.User{}).Count(&userCount)
	database.DB.Model(&models.Order{}).Count(&orderCount)

	var pendingCount int64
	database.DB.Model(&models.Article{}).Where("status = ?", "pending").Count(&pendingCount)

	var paidOrders int64
	database.DB.Model(&models.Order{}).Where("status = ?", "paid").Count(&paidOrders)

	var revenue float64
	database.DB.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&revenue)

	var vipCount int64
	database.DB.Model(&models.User{}).Where("membership_level = ?", "vip").Count(&vipCount)

	// 内容类型统计
	type ContentStat struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var contentStats []ContentStat
	database.DB.Model(&models.Article{}).Select("content_type as type, count(*) as count").Group("content_type").Scan(&contentStats)

	var couponCount int64
	database.DB.Model(&models.Coupon{}).Where("status = ?", "active").Count(&couponCount)

	var attachmentCount int64
	database.DB.Model(&models.ArticleAttachment{}).Count(&attachmentCount)

	utils.Success(c, gin.H{
		"articles":        articleCount,
		"attachmentCount": attachmentCount,
		"users":           userCount,
		"orders":          orderCount,
		"pendingArticles": pendingCount,
		"paidOrders":      paidOrders,
		"revenue":         revenue,
		"vipUsers":        vipCount,
		"activeCoupons":   couponCount,
		"contentStats":    contentStats,
	})
}

// PublishArticle 用户发布文章
func PublishArticle(c *gin.Context) {
	var req struct {
		Title         string  `json:"title" binding:"required"`
		Summary       string  `json:"summary"`
		Content       string  `json:"content" binding:"required"`
		Cover         string  `json:"cover"`
		CoverLocal    bool    `json:"coverLocal"`
		Tags          string  `json:"tags"`
		ContentType   string  `json:"contentType"`
		VideoURL      string  `json:"videoUrl"`
		Attrs         string  `json:"attrs"`
		IsPremium     bool    `json:"isPremium"`
		Price         float64 `json:"price"`
		VipPrices     string  `json:"vipPrices"`
		DownloadTitle string  `json:"downloadTitle"` // 下载模块标题（仅 contentType=download 用）
		GamePlatform  string  `json:"gamePlatform"`
		GameGenre     string  `json:"gameGenre"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error400(c, "参数错误")
		return
	}

	uid := utils.GetUserID(c)
	username := utils.GetUsername(c)

	contentType := req.ContentType
	if contentType == "" {
		contentType = "article"
	}

	article := models.Article{
		Title:         req.Title,
		Summary:       req.Summary,
		Content:       req.Content,
		Cover:         req.Cover,
		CoverLocal:    req.CoverLocal,
		Tags:          req.Tags,
		AuthorID:      uid,
		AuthorName:    username,
		Status:        "pending",
		ContentType:   contentType,
		VideoURL:      req.VideoURL,
		Attrs:         req.Attrs,
		IsPremium:     req.IsPremium,
		Price:         req.Price,
		VipPrices:     req.VipPrices,
		DownloadTitle: req.DownloadTitle,
		GamePlatform:  req.GamePlatform,
		GameGenre:     req.GameGenre,
	}

	// 自动检测游戏平台和类型（资源下载类型）
	autoDetectGameFields(&article)

	// 如果没有缩略图，自动取正文第一张图片
	if article.Cover == "" && article.Content != "" {
		article.Cover = extractFirstImage(article.Content)
	}

	// 如果没有简介，自动从正文提取摘要
	if article.Summary == "" && article.Content != "" {
		article.Summary = generateSummaryFromContent(article.Content, 200)
	}

	if err := database.DB.Create(&article).Error; err != nil {
		utils.Error500(c, "发布失败")
		return
	}

	// 发放文章先锋徽章
	var articleCount int64
	database.DB.Model(&models.Article{}).Where("author_id = ?", uid).Count(&articleCount)
	if articleCount == 1 {
		var badge models.Badge
		database.DB.Where("condition = ?", "publish_1").First(&badge)
		if badge.ID > 0 {
			database.DB.Create(&models.UserBadge{UserID: uid, BadgeID: badge.ID, EarnedAt: time.Now()})
		}
	}

	// 文章已发布，失效预缓存
	utils.ArticleCache.Delete("detail:" + strconv.FormatUint(uint64(article.ID), 10))
	utils.Success(c, article)
}

// ApproveArticle 审核文章
func ApproveArticle(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Action string `json:"action" binding:"required"` // approve / reject
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error400(c, "参数错误")
		return
	}

	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		utils.Error(c, 404, "文章不存在")
		return
	}

	if req.Action == "approve" {
		article.Status = "published"
		now := time.Now()
		article.PublishedAt = &now
	} else {
		article.Status = "rejected"
	}
	database.DB.Save(&article)

	// 通知作者
	database.DB.Create(&models.Message{
		UserID:  article.AuthorID,
		Title:   "文章审核结果",
		Content: fmt.Sprintf("您的文章《%s》%s", article.Title, map[string]string{"approve": "已通过审核", "reject": "未通过审核"}[req.Action]),
		Type:    "system",
	})

	utils.SuccessMsg(c, "操作成功")
}

func parseIntDefault(s string, def int) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if n == 0 {
		return def
	}
	return n
}

// ==================== 封面图上传与本地化 ====================

// UploadCover 上传封面图（本地文件）
func UploadCover(c *gin.Context) {
	uid := utils.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error400(c, "请选择图片文件")
		return
	}
	defer file.Close()

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true}
	if !allowedExts[ext] {
		utils.Error400(c, "仅支持 jpg/jpeg/png/gif/webp/svg 格式")
		return
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		utils.Error500(c, "读取文件失败")
		return
	}

	// 优先上传到 R2（已配置时）；失败回退本地存储
	saveName := fmt.Sprintf("%s_%d%s", time.Now().Format("20060102150405"), uid, ext)
	if url, ok := tryUploadToR2(buf, "covers", saveName, ext); ok {
		utils.Success(c, gin.H{"cover": url, "coverLocal": false})
		return
	}

	uploadDir := "data/uploads/covers"
	os.MkdirAll(uploadDir, 0755)
	savePath := filepath.Join(uploadDir, saveName)

	dst, err := os.Create(savePath)
	if err != nil {
		utils.Error500(c, "文件保存失败")
		return
	}
	defer dst.Close()
	dst.Write(buf)

	// Windows 上 filepath.Join 会产生反斜杠，URL 中必须统一为正斜杠
	coverURL := "/" + strings.ReplaceAll(savePath, "\\", "/")
	utils.Success(c, gin.H{"cover": coverURL, "coverLocal": true})
}

// UploadGeneric 通用文件上传（后台模块上传：video/comic/music/novel/attachment）
func UploadGeneric(c *gin.Context) {
	uid := utils.GetUserID(c)
	uploadType := c.PostForm("type")
	if uploadType == "" {
		uploadType = "attachments"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error400(c, "请选择文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	subDir := "attachments"
	switch uploadType {
	case "cover":
		subDir = "covers"
	case "video":
		subDir = "videos"
	case "comic":
		subDir = "comics"
	case "music":
		subDir = "music"
	case "novel":
		subDir = "novels"
	case "attachment", "attachments", "download":
		subDir = "attachments"
	default:
		subDir = "attachments"
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		utils.Error500(c, "读取文件失败")
		return
	}

	saveName := fmt.Sprintf("%s_%d%s", time.Now().Format("20060102150405"), uid, ext)
	// 优先上传到 R2（已配置时）；失败回退本地存储
	if url, ok := tryUploadToR2(buf, subDir, saveName, ext); ok {
		utils.Success(c, gin.H{
			"url":   url,
			"name":  header.Filename,
			"size":  header.Size,
			"ext":   ext,
			"type":  uploadType,
			"local": false,
		})
		return
	}

	uploadDir := filepath.Join("data/uploads", subDir)
	os.MkdirAll(uploadDir, 0755)
	savePath := filepath.Join(uploadDir, saveName)

	dst, err := os.Create(savePath)
	if err != nil {
		utils.Error500(c, "文件保存失败")
		return
	}
	defer dst.Close()
	dst.Write(buf)

	fileURL := "/" + strings.ReplaceAll(savePath, "\\", "/")
	utils.Success(c, gin.H{
		"url":   fileURL,
		"name":  header.Filename,
		"size":  header.Size,
		"ext":   ext,
		"type":  uploadType,
		"local": true,
	})
}

// LocalizeCover 将外链封面图下载到本地
func LocalizeCover(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error400(c, "请提供图片URL")
		return
	}

	// 验证URL格式
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		utils.Error400(c, "请提供有效的http/https URL")
		return
	}

	// 下载图片
	resp, err := http.Get(req.URL)
	if err != nil {
		utils.Error500(c, "下载图片失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		utils.Error500(c, "下载图片失败: HTTP "+fmt.Sprint(resp.StatusCode))
		return
	}

	// 从Content-Type或URL推断扩展名
	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "svg") || strings.Contains(contentType, "svg+xml"):
		ext = ".svg"
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		ext = ".jpg"
	default:
		// 尝试从URL获取扩展名
		urlExt := strings.ToLower(filepath.Ext(req.URL))
		if urlExt != "" && (urlExt == ".jpg" || urlExt == ".jpeg" || urlExt == ".png" || urlExt == ".gif" || urlExt == ".webp" || urlExt == ".svg") {
			ext = urlExt
		}
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Error500(c, "读取图片失败")
		return
	}

	saveName := fmt.Sprintf("%s_localized%s", time.Now().Format("20060102150405"), ext)
	// 优先上传到 R2（已配置时）；失败回退本地存储
	if url, ok := tryUploadToR2(buf, "covers", saveName, ext); ok {
		utils.Success(c, gin.H{"cover": url, "coverLocal": false, "original": req.URL})
		return
	}

	uploadDir := "data/uploads/covers"
	os.MkdirAll(uploadDir, 0755)
	savePath := filepath.Join(uploadDir, saveName)

	dst, err := os.Create(savePath)
	if err != nil {
		utils.Error500(c, "保存文件失败")
		return
	}
	defer dst.Close()

	if _, err = dst.Write(buf); err != nil {
		utils.Error500(c, "写入文件失败")
		return
	}

	coverURL := "/" + strings.ReplaceAll(savePath, "\\", "/")
	utils.Success(c, gin.H{"cover": coverURL, "coverLocal": true, "original": req.URL})
}

// tryUploadToR2 将图片字节上传到 Cloudflare R2（若已在 settings 配置）。
// 返回公共访问 URL 与是否成功；未配置或失败均返回 false（调用方回退本地存储）。
func tryUploadToR2(data []byte, subDir, saveName, ext string) (string, bool) {
	accountID := database.GetSetting("r2_account_id")
	ak := database.GetSetting("r2_access_key_id")
	sk := database.GetSetting("r2_secret_access_key")
	bucket := database.GetSetting("r2_bucket")
	if accountID == "" || ak == "" || sk == "" || bucket == "" {
		return "", false
	}
	key := subDir + "/" + saveName
	base := database.GetSetting("r2_public_url")
	if base == "" {
		base = database.GetSetting("r2_public_base")
	}
	url, err := utils.R2Upload(data, key, mimeByExt(ext), accountID, ak, sk, bucket, base)
	if err != nil {
		fmt.Printf("[R2] 上传失败，回退本地存储: %v\n", err)
		return "", false
	}
	return url, true
}

// mimeByExt 根据扩展名返回图片 Content-Type
func mimeByExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	}
	return "application/octet-stream"
}

// syncXVideoForArticle 自动同步 x.com 视频信息：补标题 + 创建 video_source
func syncXVideoForArticle(articleID uint) {
	var article models.Article
	if err := database.DB.First(&article, articleID).Error; err != nil {
		return
	}

	// 只处理 x.com / twitter.com 链接
	videoURL := strings.TrimSpace(article.VideoURL)
	re := regexp.MustCompile(`https?://(x\.com|twitter\.com)/(\w+)/status/(\d+)`)
	if !re.MatchString(videoURL) {
		return
	}

	// 调用 parseXInfo 获取推文信息
	info, err := parseXInfo(videoURL)
	if err != nil {
		return
	}

	// 1. 自动补标题（如果标题为空或看起来是默认占位标题）
	titleEmpty := strings.TrimSpace(article.Title) == ""
	if titleEmpty && info.Text != "" {
		titleText := info.Text
		// 限制标题长度
		titleRunes := []rune(titleText)
		if len(titleRunes) > 80 {
			titleText = string(titleRunes[:80])
		}
		database.DB.Model(&article).Update("title", titleText)
	}

	// 2. 自动创建 video_source（如果还没有）
	var count int64
	database.DB.Model(&models.VideoSource{}).Where("article_id = ? AND source_type = ?", articleID, "x_com").Count(&count)
	if count == 0 && info.VideoURL != "" {
		vs := models.VideoSource{
			ArticleID:  articleID,
			Title:      info.Text,
			SourceType: "x_com",
			SourceURL:  videoURL, // 存原始 x.com 链接，播放时通过 proxy-x 实时解析
			Thumbnail:  info.Poster,
			Sort:       0,
			Status:     "active",
		}
		result := database.DB.Create(&vs)
		if result.Error == nil {
			// 异步下载视频到本地缓存
			go CacheXVideosForArticle(articleID)
		}
	}

	// 3. 自动填充封面图（如果文章没有封面且 API 返回了封面）
	if strings.TrimSpace(article.Cover) == "" && info.Poster != "" {
		database.DB.Model(&article).Update("cover", info.Poster)
	}
}

// extractFirstImage 从 HTML/Markdown 内容中提取第一张图片 URL
func extractFirstImage(content string) string {
	// 匹配 Markdown 图片: ![alt](url) — 支持绝对 URL 和相对路径
	mdRe := regexp.MustCompile(`!\[.*?\]\((https?://[^\s)]+|/[^\s)]+)\)`)
	if matches := mdRe.FindStringSubmatch(content); len(matches) >= 2 {
		return matches[1]
	}
	// 匹配 HTML <img> 标签 — 支持绝对 URL 和相对路径
	htmlRe := regexp.MustCompile(`<img[^>]+src=["'](https?://[^"'\s>]+|/[^"'\s>]+)["']`)
	if matches := htmlRe.FindStringSubmatch(content); len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// detectGamePlatform 从标题自动识别游戏平台
// 返回值:
//   - "both"   电脑+手机双端
//   - "mobile" 仅手机/安卓
//   - "pc"     仅电脑/单机
//
// 判定规则（按从强到弱匹配）:
//   1) 同时含"电脑"与"手机/安卓/移动/ios/apk" → both
//   2) 仅含"手机/安卓/移动/ios/apk"          → mobile
//   3) 其余                                  → pc（兜底默认）
func detectGamePlatform(title string) string {
	// 电脑关键字
	pcKws := []string{"电脑", "PC版", "PC单机", "PC+", "pc版"}
	// 手机关键字（涵盖安卓/iOS/移动版/APK）
	mobileKws := []string{"手机", "安卓", "移动版", "APK", "apk", "ios版", "iOS版"}

	hasPC := false
	for _, k := range pcKws {
		if strings.Contains(title, k) {
			hasPC = true
			break
		}
	}
	hasMobile := false
	for _, k := range mobileKws {
		if strings.Contains(title, k) {
			hasMobile = true
			break
		}
	}

	if hasPC && hasMobile {
		return "both"
	}
	if hasMobile {
		return "mobile"
	}
	return "pc"
}

// detectGameGenre 从标题自动识别游戏类型
// 标题含 SLG/RPG/ACT/ADV/HTML → 返回匹配的类型，逗号分隔
func detectGameGenre(title string) string {
	upperTitle := strings.ToUpper(title)
	var genres []string
	genreKeywords := []string{"SLG", "RPG", "ACT", "ADV", "HTML"}
	for _, g := range genreKeywords {
		if strings.Contains(upperTitle, g) {
			genres = append(genres, g)
		}
	}
	return strings.Join(genres, ",")
}

// autoDetectGameFields 自动检测并填充游戏平台和类型字段（仅资源下载类型）
func autoDetectGameFields(article *models.Article) {
	if article.ContentType != "download" {
		return
	}
	// 如果用户未手动设置，则自动检测
	if article.GamePlatform == "" {
		article.GamePlatform = detectGamePlatform(article.Title)
	}
	if article.GameGenre == "" {
		article.GameGenre = detectGameGenre(article.Title)
	}
}

// AdminFixThumbnails 批量修复无缩略图的文章，从正文提取第一张图片
func AdminFixThumbnails(c *gin.Context) {
	var articles []models.Article
	database.DB.Where("cover = '' OR cover IS NULL").Where("content != ''").Find(&articles)

	fixed := 0
	for _, a := range articles {
		cover := extractFirstImage(a.Content)
		if cover != "" {
			database.DB.Model(&a).Update("cover", cover)
			fixed++
		}
	}
	utils.Success(c, gin.H{"total": len(articles), "fixed": fixed})
}

// ReportDeadLink 资源失效反馈（发送邮件通知管理员）
func ReportDeadLink(c *gin.Context) {
	articleID := c.Param("id")
	uid := utils.GetUserID(c)

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error400(c, "请填写反馈内容")
		return
	}

	// 获取文章信息
	var article models.Article
	articleTitle := "未知文章"
	if err := database.DB.First(&article, articleID).Error; err == nil {
		articleTitle = article.Title
	}

	// 获取用户信息
	var user models.User
	userName := "匿名用户"
	if uid > 0 {
		if err := database.DB.First(&user, uid).Error; err == nil {
			userName = user.Username
		}
	}

	// 获取管理员邮箱
	adminEmail := database.GetSetting("site_email")
	if adminEmail == "" {
		// 如果没有配置管理员邮箱，也记录到数据库
		database.DB.Create(&models.Message{
			UserID:  1, // 发送给管理员（默认 admin ID=1）
			Title:   "资源失效反馈",
			Content: fmt.Sprintf("文章《%s》(ID:%s) 被 %s 报告链接失效：\n%s", articleTitle, articleID, userName, req.Message),
			Type:    "system",
		})
		utils.SuccessMsg(c, "感谢反馈，我们会尽快处理！")
		return
	}

	// 异步发送邮件通知管理员
	go func() { defer func() { _ = recover() }()
		subject := fmt.Sprintf("【TechLife Blog】资源失效反馈 - 《%s》", articleTitle)
		body := fmt.Sprintf(
			"您好，管理员：\n\n"+
				"收到一条资源失效反馈，详情如下：\n\n"+
				"📄 文章：《%s》(ID: %s)\n"+
				"👤 报告人：%s\n"+
				"📝 反馈内容：\n%s\n\n"+
				"请尽快登录后台检查并修复资源链接。\n\n"+
				"— TechLife Blog 系统通知",
			articleTitle, articleID, userName, req.Message,
		)
		if err := sendSMTPMail(adminEmail, subject, body); err != nil {
			fmt.Printf("[失效反馈] 邮件发送失败: %v\n", err)
		}
	}()

	// 同时存一条站内消息给管理员
	database.DB.Create(&models.Message{
		UserID:  1,
		Title:   "资源失效反馈",
		Content: fmt.Sprintf("文章《%s》(ID:%s) 被 %s 报告链接失效：\n%s", articleTitle, articleID, userName, req.Message),
		Type:    "system",
	})

	utils.SuccessMsg(c, "感谢反馈，我们会尽快处理！")
}

// AdminResyncSummaries 批量重新生成文章的简介（从正文提取）
// mode=empty（默认）: 仅填充空摘要；mode=all: 覆盖所有摘要；mode=jianjie: 仅重算含「简介」标记的文章，用简介后的文字覆盖
func AdminResyncSummaries(c *gin.Context) {
	mode := c.DefaultQuery("mode", "empty")
	var articles []models.Article
	if mode == "jianjie" {
		database.DB.Where("content LIKE ?", "%简介%").Find(&articles)
	} else if mode == "all" {
		database.DB.Where("content != ''").Find(&articles)
	} else {
		database.DB.Where("content != ''").Where("summary = '' OR summary IS NULL").Find(&articles)
	}

	updated := 0
	for _, a := range articles {
		summary := generateSummaryFromContent(a.Content, 200)
		if summary != "" {
			database.DB.Model(&a).Update("summary", summary)
			updated++
		}
	}
	utils.Success(c, gin.H{"total": len(articles), "updated": updated, "mode": mode})
}

// generateSummaryFromContent 从正文提取简介：优先取「简介」标记后的文字；无标记则退回纯文本摘要
func generateSummaryFromContent(content string, maxLen int) string {
	if content == "" {
		return ""
	}
	// 先剥离 seo干扰码（旧版隐藏关键词 span + 新版不可见字符），避免污染文章简介 / GEO 输出
	content = cleanInterferenceCode(content)
	// 优先从「简介」标记后提取（游戏/资源类正文常见「游戏简介」「xxx简介：」）
	if j := strings.Index(content, "简介"); j >= 0 {
		if s := extractSummaryAfterJianjie(content[j:], maxLen); s != "" {
			return s
		}
	}
	return extractPlainSummary(content, maxLen)
}

// extractSummaryAfterJianjie tail 以「简介」开头，返回其后的简介文字（截断到 maxLen）
func extractSummaryAfterJianjie(tail string, maxLen int) string {
	// 块级标签换成空格，便于按段落边界切分
	text := regexp.MustCompile(`(?i)</(p|div|br|tr|li|h[1-6]|blockquote)>`).ReplaceAllString(tail, " ")
	// 移除其余 HTML 标签
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	// 去掉开头的「简介」字样
	text = strings.TrimPrefix(text, "简介")
	// 去掉起始的标点与空白（：:。.、 等）
	text = strings.TrimLeft(text, "：:。.、 \t\n\r")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	// 若后方紧跟新的小节（【xxx】），只取到小节前，避免把后续下载信息带进简介
	if i := strings.Index(text, "【"); i > 12 {
		text = text[:i]
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > maxLen {
		text = string(runes[:maxLen])
	}
	return strings.TrimSpace(text)
}

// extractPlainSummary 从 HTML/Markdown 内容提取纯文本摘要（旧逻辑，作为无「简介」标记时的退回）
func extractPlainSummary(content string, maxLen int) string {
	// 移除 Markdown 图片语法 ![alt](url)
	text := regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`).ReplaceAllString(content, "")
	// 移除 Markdown 链接语法 [text](url) → 保留 text
	text = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`).ReplaceAllString(text, "$1")
	// 移除 HTML 标签
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")
	// 移除 Markdown 代码块
	text = regexp.MustCompile("```[^`]*```").ReplaceAllString(text, "")
	// 移除行内代码
	text = regexp.MustCompile("`[^`]*`").ReplaceAllString(text, "")
	// 移除 Markdown 标题标记
	text = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(text, "")
	// 移除 Markdown 引用
	text = regexp.MustCompile(`(?m)^>\s*`).ReplaceAllString(text, "")
	// 移除 Markdown 列表标记
	text = regexp.MustCompile(`(?m)^[\*\-+]\s+`).ReplaceAllString(text, "")
	// 移除 Markdown 水平线
	text = regexp.MustCompile(`(?m)^-{3,}$`).ReplaceAllString(text, "")
	// 移除多余空白
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// 截取指定长度
	runes := []rune(text)
	if len(runes) > maxLen {
		text = string(runes[:maxLen]) + "..."
	}
	return text
}
