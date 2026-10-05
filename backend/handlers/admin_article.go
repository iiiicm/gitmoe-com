package handlers

import (
	"log"
	"sort"

	"techlife-blog/database"
	"techlife-blog/models"
	"techlife-blog/utils"

	"github.com/gin-gonic/gin"
)

// articleUpdatableFields 文章更新字段白名单。
//
// 为什么要白名单：handlers/article.go 的 UpdateArticle 用 ShouldBindJSON 绑定成
// map[string]interface{} 后直接 Updates()。实测（gorm v1.31.2 + SQLite）两个坑：
//  1. Updates(map) 遇到「不存在的列」会让整条 UPDATE 报 "no such column"，
//     同一批次里的好字段也全部写不进去；而旧代码没有接收 error，前端拿到的是
//     「保存成功」但数据没改 —— 静默假成功。
//  2. 前端可以顺带传 views / author_id / id 等敏感字段被直接改写。
//
// 因此绑定后按本白名单过滤：不在表内的 key 一律静默丢弃（不报错，保证对现有前端
// 完全向后兼容），再交给 Updates，并接收 error 返回 500。
//
// 维护约定：
//   - key 同时收录 camelCase（前端 JSON / Go json tag）与 snake_case（DB 列名），
//     两种写法都能通过（GORM 的 Updates(map) 两种都能命中）。
//   - 必须完整覆盖现有后台 app/articles/page.jsx 的 save() payload 的 17 个字段。
//   - 明确不收录：id / views / like_count / author_id / created_at / updated_at /
//     deleted_at / category_slug / category_name（后两者是 gorm:"-" 注入字段，无 DB 列）。
var articleUpdatableFields = map[string]struct{}{
	// ---- 现有前端正在传的 17 个字段（缺一即线上后台不可用）----
	"title": {}, "summary": {}, "content": {}, "cover": {}, "tags": {},
	"contentType": {}, "content_type": {},
	"categoryId": {}, "category_id": {},
	"status": {}, "price": {},
	"originalPrice": {}, "original_price": {},
	"isPremium": {}, "is_premium": {},
	"membersOnly": {}, "members_only": {},
	"vipPrices": {}, "vip_prices": {},
	"downloadTitle": {}, "download_title": {},
	"gamePlatform": {}, "game_platform": {},
	"gameGenre": {}, "game_genre": {},
	"videoUrl": {}, "video_url": {},

	// ---- 其它既有可写列（宽松收录，便于后台其它入口扩展）----
	"coverLocal": {}, "cover_local": {},
	"menuId": {}, "menu_id": {},
	"authorName": {}, "author_name": {},
	"payModule": {}, "pay_module": {},
	"attrs":       {},
	"scheduledAt": {}, "scheduled_at": {},
	"releaseStatus": {}, "release_status": {},
	"publishedAt": {}, "published_at": {},
	"externalRef": {}, "external_ref": {},

	// ---- 本次新增的 28 + 1 列 ----
	"subtitle":   {},
	"sourceType": {}, "source_type": {},
	"sourceName": {}, "source_name": {},
	"sourceUrl": {}, "source_url": {},
	"isFeatured": {}, "is_featured": {},
	"allowComment": {}, "allow_comment": {},
	"serialStatus": {}, "serial_status": {},
	"region":          {},
	"language":        {},
	"resourceVersion": {}, "resource_version": {},
	"requirement":         {},
	"resourceSizeDisplay": {}, "resource_size_display": {},
	"publisher":  {},
	"isOfficial": {}, "is_official": {},
	"resourceUpdatedAt": {}, "resource_updated_at": {},
	"durationMinutes": {}, "duration_minutes": {},
	"episodeTotal": {}, "episode_total": {},
	"releaseYear": {}, "release_year": {},
	"releaseDate": {}, "release_date": {},
	"cast":           {},
	"director":       {},
	"qualityOptions": {}, "quality_options": {},
	"illustrator":  {},
	"chapterTotal": {}, "chapter_total": {},
	"translationStatus": {}, "translation_status": {},
	"firstPlatform": {}, "first_platform": {},
	"wordCount": {}, "word_count": {},
	"contentFormat": {}, "content_format": {},
}

// filterArticleUpdates 按白名单过滤文章更新字段。
// 返回允许写入的字段 map 与被丢弃的未知 key（仅用于日志，不回给前端）。
func filterArticleUpdates(in map[string]interface{}) (map[string]interface{}, []string) {
	out := make(map[string]interface{}, len(in))
	dropped := make([]string, 0)
	for k, v := range in {
		if _, ok := articleUpdatableFields[k]; ok {
			out[k] = v
		} else {
			dropped = append(dropped, k)
		}
	}
	if len(dropped) > 1 {
		sort.Strings(dropped)
	}
	return out, dropped
}

// GetArticleDetailAdmin 后台编辑用文章详情：一次性返回主表 + 附件 + 视频源 + 章节 + SEO。
//
// 路由：GET /api/admin/article/:id（★ 单数，见设计文档 §0.3）
// 不复用公开 GetArticle：后者会自增 views、发积分、受 hasAccess 影响。
func GetArticleDetailAdmin(c *gin.Context) {
	id := c.Param("id")

	var article models.Article
	if err := database.DB.First(&article, id).Error; err != nil {
		utils.Error(c, 404, "文章不存在")
		return
	}

	// 子资源一律返回 [] 而非 null（前端 .map 遇 null 会崩）
	attachments := make([]models.ArticleAttachment, 0)
	if err := database.DB.Where("article_id = ?", article.ID).
		Order("sort asc, id asc").Find(&attachments).Error; err != nil {
		log.Printf("[WARN] GetArticleDetailAdmin 查询 attachments 失败 id=%s: %v", id, err)
	}

	videoSources := make([]models.VideoSource, 0)
	if err := database.DB.Where("article_id = ?", article.ID).
		Order("episode asc, sort asc, id asc").Find(&videoSources).Error; err != nil {
		log.Printf("[WARN] GetArticleDetailAdmin 查询 videoSources 失败 id=%s: %v", id, err)
	}

	chapters := make([]models.Chapter, 0)
	if err := database.DB.Where("content_id = ?", article.ID).
		Order("chapter_order asc, id asc").Find(&chapters).Error; err != nil {
		log.Printf("[WARN] GetArticleDetailAdmin 查询 chapters 失败 id=%s: %v", id, err)
	}

	// SEO 用 First + 忽略 ErrRecordNotFound：无记录时输出 null（前端按 null 分支处理）
	var seo *models.ArticleSEO
	var seoRow models.ArticleSEO
	if err := database.DB.Where("article_id = ?", article.ID).First(&seoRow).Error; err == nil {
		seo = &seoRow
	}

	utils.Success(c, gin.H{
		"article":      article,
		"attachments":  attachments,
		"videoSources": videoSources,
		"chapters":     chapters,
		"seo":          seo,
	})
}
