// article-editor 共享常量（列表页与编辑器共用，单一数据源）。
// 修改后请注意：app/articles/page.jsx 与编辑壳都从这里 import，避免两份漂移。

// music 下线总开关：false = 后台完全不提供音乐入口（存量 music 文章只读/隐藏徽标）。
export const MUSIC_MODULE_ENABLED = false

export const CONTENT_TYPES = [
  { v: 'article', label: '新闻模块', desc: '普通文章' },
  { v: 'download', label: '下载模块', desc: '付费下载' },
  { v: 'video', label: '视频模块', desc: '视频解析' },
  { v: 'comic', label: '漫画模块', desc: '漫画阅读' },
  { v: 'novel', label: '小说模块', desc: '小说阅读' },
  { v: 'music', label: '音乐模块', desc: '免费展示' },
]

// 列表/下拉/编辑页一律引用此数组（自动剔除下线的 music）。
export const VISIBLE_CONTENT_TYPES = CONTENT_TYPES.filter(
  (t) => MUSIC_MODULE_ENABLED || t.v !== 'music'
)

// 付费模块：含注册用户价格 + 各会员价 + 仅会员开关。控制 PaidMetabox 显隐。
export const PAID_MODULES = ['download', 'video', 'comic', 'novel']

export const MEMBERS_ONLY_LABEL = {
  download: '仅会员可下载',
  video: '仅会员可看',
  comic: '仅会员可看',
  novel: '仅会员可看',
}

export const STATUS_OPTS = [
  { v: 'published', label: '已发布' },
  { v: 'pending', label: '待审核' },
  { v: 'draft', label: '草稿' },
]

// ===== 下载资源链接类型 / 网盘平台 =====
export const LINK_TYPES = [
  { v: 'netdisk', label: '网盘链接' },
  { v: 'external', label: '外部直链' },
]
export const PLATFORMS = [
  { v: 'baidu', label: '百度网盘' },
  { v: 'quark', label: '夸克网盘' },
  { v: 'ali', label: '阿里云盘' },
  { v: '115', label: '115网盘' },
  { v: 'other', label: '其他' },
]

// ===== 视频源类型 =====
export const VIDEO_TYPES = [
  { v: 'mp4', label: 'MP4 直链' },
  { v: 'm3u8', label: 'M3U8' },
  { v: 'x_com', label: 'X.com / Twitter' },
  { v: 'youtube', label: 'YouTube' },
  { v: 'bilibili', label: 'Bilibili' },
]

// ===== 游戏类型 / 设备（与前端分类页 GENRE_ORDER 一致）=====
export const GENRE_ORDER = ['SLG', 'RPG', 'ADV', 'ACT', 'HTML', '其他']
export const PLATFORM_ORDER = ['pc', 'mobile', 'both']
export const PLATFORM_LABELS = { pc: '电脑', mobile: '手机', both: '电脑+手机' }

// ===== 本次新增类型字段枚举 =====
export const SOURCE_TYPES = [
  { v: 'original', label: '原创' },
  { v: 'reprint', label: '转载' },
  { v: 'compile', label: '整理/汇编' },
]
export const SERIAL_STATUS = [
  { v: '', label: '未知/不确定' },
  { v: 'ongoing', label: '连载中' },
  { v: 'completed', label: '已完结' },
  { v: 'paused', label: '已暂停' },
]
export const REGIONS = [
  { v: '', label: '不限/未知' },
  { v: 'cn', label: '国产' },
  { v: 'jp', label: '日本' },
  { v: 'kr', label: '韩国' },
  { v: 'us', label: '欧美' },
  { v: 'other', label: '其他' },
]
export const LANGUAGES = [
  { v: '', label: '未设置' },
  { v: 'zh-CN', label: '简体中文' },
  { v: 'zh-TW', label: '繁体中文' },
  { v: 'en', label: '英文' },
  { v: 'ja', label: '日文' },
  { v: 'ko', label: '韩文' },
  { v: 'multi', label: '多语言' },
  { v: 'other', label: '其他' },
]
export const TRANSLATION_STATUS = [
  { v: '', label: '未设置' },
  { v: 'official_zh', label: '官方中文' },
  { v: 'fansub', label: '民间汉化' },
  { v: 'raw', label: '生肉/原版' },
  { v: 'unknown', label: '未知' },
]

// 安全解析 vipPrices（可能为 JSON 字符串或对象）。
export function parseVipPrices(raw) {
  if (!raw) return {}
  if (typeof raw === 'object') return raw
  try {
    const o = JSON.parse(raw)
    return o && typeof o === 'object' ? o : {}
  } catch (e) {
    return {}
  }
}
