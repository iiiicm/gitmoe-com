// 唯一保存请求体构造函数：严格对齐旧 page.jsx save() 的 17 个字段，并扩展 28 个新列。
// 子条目（附件/视频源/章节）不进 payload，仍走各自逐条 CRUD。
import { PAID_MODULES } from './constants'

function toIsoDate(s) {
  if (!s) return null
  if (/^\d{4}-\d{2}-\d{2}/.test(s)) {
    return s.length >= 10 ? s.slice(0, 10) + 'T00:00:00Z' : null
  }
  return null
}

export function buildSavePayload(form) {
  const isPaid = PAID_MODULES.includes(form.contentType)
  const price = Number(form.price) || 0
  const vipObj = isPaid ? (form.vipPrices || {}) : {}
  const memberPrices = Object.values(vipObj).map((n) => Number(n) || 0).filter((n) => n > 0)
  const isPremium = isPaid && (price > 0 || memberPrices.length > 0)

  return {
    // ---- 现有 17 字段 ----
    title: form.title,
    summary: form.summary,
    content: form.content,
    contentFormat: 'html', // 新编辑器一律产出 HTML
    cover: form.cover,
    tags: form.tags,
    contentType: form.contentType,
    categoryId: form.categoryId ? Number(form.categoryId) : null,
    status: form.status,
    releaseStatus: form.releaseStatus,
    scheduledAt: form.scheduledAt || null,
    price,
    originalPrice: Number(form.originalPrice) || 0,
    isPremium,
    membersOnly: isPaid ? !!form.membersOnly : false,
    vipPrices: JSON.stringify(vipObj),
    downloadTitle: (form.downloadTitle || '').trim(),
    gamePlatform: form.gamePlatform || '',
    gameGenre: form.gameGenre || '',
    videoUrl: form.contentType === 'video' ? form.videoUrl : '',
    authorName: form.authorName || '',
    // ---- 本次新增 28 列 ----
    subtitle: form.subtitle || '',
    sourceType: form.sourceType || 'original',
    sourceName: form.sourceName || '',
    sourceUrl: form.sourceUrl || '',
    isFeatured: !!form.isFeatured,
    allowComment: !!form.allowComment,
    serialStatus: form.serialStatus || '',
    region: form.region || '',
    language: form.language || '',
    resourceVersion: form.resourceVersion || '',
    requirement: form.requirement || '',
    resourceSizeDisplay: form.resourceSizeDisplay || '',
    publisher: form.publisher || '',
    isOfficial: !!form.isOfficial,
    resourceUpdatedAt: toIsoDate(form.resourceUpdatedAt),
    durationMinutes: Number(form.durationMinutes) || 0,
    episodeTotal: Number(form.episodeTotal) || 0,
    releaseYear: Number(form.releaseYear) || 0,
    releaseDate: toIsoDate(form.releaseDate),
    cast: form.cast || '',
    director: form.director || '',
    qualityOptions: form.qualityOptions || '',
    illustrator: form.illustrator || '',
    chapterTotal: Number(form.chapterTotal) || 0,
    translationStatus: form.translationStatus || '',
    firstPlatform: form.firstPlatform || '',
    wordCount: Number(form.wordCount) || 0,
  }
}
