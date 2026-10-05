import { notFound } from 'next/navigation'
import { articleApi, categoryApi } from '../lib/api'
import { renderArticleBody } from '../lib/render'
import { formatDate } from '../lib/format'
import { buildDetailMetadata } from '../lib/seo'
import DetailJsonLd from './DetailJsonLd'
import ArticleInteractions from '../app/article/[id]/ArticleInteractions'
import ArticleTitle from './ArticleTitle'
import ArticleBottomToolbar from './ArticleBottomToolbar'
import ArticleBreadcrumb from './ArticleBreadcrumb'
import ArticleContentGate from './ArticleContentGate'

// 共享的文章详情元数据生成（供 /article/[id] 与 /:slug/:id.html 复用）
// 返回完整 SEO metadata：title / description / canonical / OpenGraph / JSON-LD
export async function buildArticleMetadata(id) {
  try {
    const res = await articleApi.detail(id)
    if (res.code === 0 && res.data?.article) {
      // 游戏资源（categorySlug === 'games'）不在网页端展现：直接 404（游戏仅在 Windows / 安卓 客户端展现）
      if (res.data.article.categorySlug === 'games') {
        notFound()
      }
      return buildDetailMetadata(res.data.article, 'article')
    }
  } catch (e) {
    /* ignore */
  }
  return { title: '文章未找到 - Gitmoe' }
}

// 服务端组件：请求时取文章并服务端渲染正文 HTML（进入 SSR 源码，SEO 友好）
export default async function ArticleDetail({ id }) {
  let res
  try {
    res = await articleApi.detail(id)
  } catch (e) {
    res = { code: -1, message: '网络异常，文章加载失败' }
  }

  if (res.code !== 0 || !res.data?.article) {
    return (
      <div className="container section">
        <div className="article-error-state">
          <i className="fa-solid fa-circle-exclamation" style={{ fontSize: 48, color: '#f59e0b', marginBottom: 16 }} />
          <h3>文章加载失败</h3>
          <p>{res.message || '文章不存在'}</p>
        </div>
      </div>
    )
  }

  const article = res.data.article
  // 游戏资源（categorySlug === 'games'）不在网页端展现：直接 404（游戏仅在 Windows / 安卓 客户端展现）
  if (article.categorySlug === 'games') {
    notFound()
  }
  const hasAccess = res.data.hasAccess
  const locked = res.data.locked
  const bodyHtml = renderArticleBody(article.content, article.contentFormat)

  // 面包屑：用 categoryId 匹配分类名称与 slug（公开接口，失败则降级为空）
  let categoryName = ''
  let categorySlug = ''
  try {
    const catRes = await categoryApi.list()
    const cats = catRes?.data || catRes || []
    const cat = (Array.isArray(cats) ? cats : []).find((c) => String(c.id) === String(article.categoryId))
    if (cat) {
      categoryName = cat.name || ''
      categorySlug = cat.slug || ''
    }
  } catch (e) {
    /* 分类获取失败不影响正文渲染 */
  }

  return (
    <div className="container section">
        {/* 结构化数据：Article + BreadcrumbList（GEO/SEO） */}
        <DetailJsonLd article={article} type="article" categoryName={categoryName} categorySlug={categorySlug} />

        {/* 面包屑：首页 - 分类名称 - 正文（视频模块不加） */}
        <ArticleBreadcrumb categoryName={categoryName} categorySlug={categorySlug} />

        {/* 浅色背景块：从标题到正文结束 */}
        <div className="article-card">
          <ArticleTitle title={article.title} className="article-detail-title" />
          <div className="article-meta">
            <span>{article.authorName}</span>
            <span>{formatDate(article.publishedAt || article.createdAt)}</span>
            <span>{article.views} 阅读</span>
            {article.isPremium && article.contentType !== 'download' && (
              <span className="tag">付费文章</span>
            )}
            {article.membersOnly && (
              <span className="tag">会员专享</span>
            )}
          </div>

          {locked && (
            /* 受限内容且无权访问：正文仍可见，仅顶部加一条付费提示，下载资源在下方底部工具条付费解锁 */
            <div className="article-paywall-hint">
              <i className="fa-solid fa-lock" />
              <span>{article.membersOnly
                ? '本文为会员专享，开通会员后可解锁下方付费下载资源。'
                : '本文为付费内容，购买或开通会员后可解锁下方付费下载资源。'}</span>
            </div>
          )}

          {/* 服务端渲染的正文（已在 HTML 源码中，利于 SEO）；正文对用户始终可见。
              通过 ArticleContentGate 拦截「相关游戏」卡片点击，网页端弹「下载客户端」窗。 */}
          <ArticleContentGate html={bodyHtml} />
        </div>

        {/* 交互部分（视频/标签/回到顶部）下沉到客户端组件 */}
        <ArticleInteractions id={id} article={article} hasAccess={hasAccess} />

        {/* 底部工具条：仅下载模块（content_type='download'）展示，含开通会员 / 付费下载 / 打开客户端 */}
        {article.contentType === 'download' && (
          <ArticleBottomToolbar id={id} article={article} hasAccess={hasAccess} />
        )}
    </div>
  )
}
