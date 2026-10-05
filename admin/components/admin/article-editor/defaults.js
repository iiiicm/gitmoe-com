// 表单默认值 + 后端数据 → form 映射。所有 key 与 Go json tag 同名。
import { parseVipPrices } from './constants'
import { isHtmlContent } from '../../../lib/content'

function toDateInput(iso) {
  if (!iso) return ''
  const s = String(iso)
  if (/^\d{4}-\d{2}-\d{2}/.test(s)) return s.slice(0, 10)
  const d = new Date(s)
  if (isNaN(d.getTime())) return ''
  return d.toISOString().slice(0, 10)
}

export function emptyForm() {
  return {
    id: null,
    title: '',
    content: '',
    contentFormat: 'html',
    summary: '',
    cover: '',
    coverLocal: false,
    tags: '',
    categoryId: '',
    contentType: 'article',
    status: 'published',
    releaseStatus: 'published',
    scheduledAt: null,
    price: 0,
    originalPrice: 0,
    vipPrices: {},
    isPremium: false,
    membersOnly: false,
    downloadTitle: '',
    gamePlatform: '',
    gameGenre: '',
    videoUrl: '',
    authorName: '',
    // ---- 本次新增 28 列 ----
    subtitle: '',
    sourceType: 'original',
    sourceName: '',
    sourceUrl: '',
    isFeatured: false,
    allowComment: true,
    serialStatus: '',
    region: '',
    language: '',
    resourceVersion: '',
    requirement: '',
    resourceSizeDisplay: '',
    publisher: '',
    isOfficial: false,
    resourceUpdatedAt: null,
    durationMinutes: 0,
    episodeTotal: 0,
    releaseYear: 0,
    releaseDate: null,
    cast: '',
    director: '',
    qualityOptions: '',
    illustrator: '',
    chapterTotal: 0,
    translationStatus: '',
    firstPlatform: '',
    wordCount: 0,
  }
}

export function formFromArticle(a) {
  if (!a) return emptyForm()
  return {
    id: a.id ?? null,
    title: a.title || '',
    content: a.content || '',
    contentFormat: a.contentFormat || (isHtmlContent(a.content) ? 'html' : ''),
    summary: a.summary || '',
    cover: a.cover || '',
    coverLocal: !!a.coverLocal,
    tags: a.tags || '',
    categoryId: a.categoryId != null ? String(a.categoryId) : '',
    contentType: a.contentType || 'article',
    status: a.status || 'published',
    releaseStatus: a.releaseStatus || 'published',
    scheduledAt: a.scheduledAt || null,
    price: a.price || 0,
    originalPrice: a.originalPrice || 0,
    vipPrices: parseVipPrices(a.vipPrices),
    isPremium: !!a.isPremium,
    membersOnly: !!a.membersOnly,
    downloadTitle: a.downloadTitle || '',
    gamePlatform: a.gamePlatform || '',
    gameGenre: a.gameGenre || '',
    videoUrl: a.videoUrl || '',
    authorName: a.authorName || '',
    subtitle: a.subtitle || '',
    sourceType: a.sourceType || 'original',
    sourceName: a.sourceName || '',
    sourceUrl: a.sourceUrl || '',
    isFeatured: !!a.isFeatured,
    allowComment: a.allowComment !== false,
    serialStatus: a.serialStatus || '',
    region: a.region || '',
    language: a.language || '',
    resourceVersion: a.resourceVersion || '',
    requirement: a.requirement || '',
    resourceSizeDisplay: a.resourceSizeDisplay || '',
    publisher: a.publisher || '',
    isOfficial: !!a.isOfficial,
    resourceUpdatedAt: toDateInput(a.resourceUpdatedAt),
    durationMinutes: a.durationMinutes || 0,
    episodeTotal: a.episodeTotal || 0,
    releaseYear: a.releaseYear || 0,
    releaseDate: toDateInput(a.releaseDate),
    cast: a.cast || '',
    director: a.director || '',
    qualityOptions: a.qualityOptions || '',
    illustrator: a.illustrator || '',
    chapterTotal: a.chapterTotal || 0,
    translationStatus: a.translationStatus || '',
    firstPlatform: a.firstPlatform || '',
    wordCount: a.wordCount || 0,
  }
}
