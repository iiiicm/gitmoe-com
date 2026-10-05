// 后台管理 API 封装：复用 lib/api 的鉴权请求函数（Bearer token 取自 localStorage）。
import { apiGet, authGet, apiPost, apiPut, apiDelete, apiPostForm, getToken } from './api'

// 鉴权头助手：缺失 token 时直接拦截并跳转登录，避免发送非法的 "Bearer " 空头。
function authHeaders() {
  const t = getToken()
  if (!t) {
    if (typeof window !== 'undefined') {
      window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname || '/')
    }
    const err = new Error('未登录')
    err.status = 401
    throw err
  }
  return { Authorization: 'Bearer ' + t }
}

// 上传会员体系图标（组图标 / 等级图标，支持 ico/png/jpg/jpeg/gif/webp/svg），需鉴权 + FormData
async function uploadMembershipIcon(file) {
  const base =
    typeof window === 'undefined'
      ? (process.env.API_BASE_INTERNAL || 'http://localhost:8080/api')
      : (process.env.NEXT_PUBLIC_API_BASE || '/api')
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(base + '/admin/upload-membership-icon', {
    method: 'POST',
    headers: authHeaders(),
    body: form,
  })
  if (!res.ok) {
    let data = {}
    try { data = await res.json() } catch (e) { /* ignore */ }
    const err = new Error('上传失败 ' + res.status)
    err.status = res.status
    err.data = data
    throw err
  }
  return res.json()
}


const dashboard = {
  overview: () => authGet('/admin/dashboard'),
  today: () => authGet('/admin/dashboard/today'),
}

const articles = {
  get: (id) => authGet('/admin/article/' + id),
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/articles' + (q ? '?' + q : ''))
  },
  create: (body) => apiPost('/admin/articles', body),
  update: (id, body) => apiPut('/admin/articles/' + id, body),
  remove: (id) => apiDelete('/admin/articles/' + id),
  batchDelete: (ids) => apiPost('/admin/articles/batch-delete', { ids }),
  batchChangeCategory: (ids, categoryId) =>
    apiPost('/admin/articles/batch-change-category', { ids, categoryId }),
  batchChangeStatus: (ids, status) =>
    apiPost('/admin/articles/batch-change-status', { ids, status }),
  batchChangeModule: (ids, type) =>
    apiPost('/admin/articles/batch-change-module', { ids, type }),
  batchSetPrice: (ids, payload) =>
    apiPost('/admin/articles/batch-set-price', { ids, ...payload }),
  batchChangeGenre: (ids, genre) =>
    apiPost('/admin/articles/batch-change-genre', { ids, genre }),
  batchAutoMatchGenre: (ids, force = false) =>
    apiPost('/admin/articles/batch-auto-match-genre', { ids, force }),
  uploadCover: (formData) => apiPostForm('/admin/upload-cover', formData),
  uploadGeneric: (formData) => apiPostForm('/admin/upload-generic', formData),
}

const categories = {
  list: () => authGet('/admin/categories'),
  create: (body) => apiPost('/admin/categories', body),
  update: (id, body) => apiPut('/admin/categories/' + id, body),
  remove: (id) => apiDelete('/admin/categories/' + id),
}

// 单页（独立页面）管理：GET/POST /admin/pages、PUT/DELETE /admin/pages/:id
const pages = {
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/pages' + (q ? '?' + q : ''))
  },
  create: (body) => apiPost('/admin/pages', body),
  update: (id, body) => apiPut('/admin/pages/' + id, body),
  remove: (id) => apiDelete('/admin/pages/' + id),
}

// 文章评论管理：列表 + 改状态（published/hidden）+ 单删 / 批删
const comments = {
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/comments' + (q ? '?' + q : ''))
  },
  // status 仅允许 published / hidden（后端校验）
  setStatus: (id, status) => apiPut('/admin/comments/' + id + '/status', { status }),
  remove: (id) => apiDelete('/admin/comments/' + id),
  batchDelete: (ids) => apiPost('/admin/comments/batch-delete', { ids }),
}

// 标签管理（无独立表：聚合 articles.tags 逗号串）
const tags = {
  list: () => authGet('/admin/tags'),
  rename: (oldName, newName) => apiPost('/admin/tags/rename', { oldName, newName }),
  batchDelete: (names) => apiPost('/admin/tags/batch-delete', { names }),
}

// GEO 概览（面向 AI 引擎 / 大模型）：产出物（llms.txt / feed.xml / sitemap / robots.txt）
// 的体积·条目数·状态聚合，含爬虫抓取快照。推送动作复用 seo.push。
const geo = {
  status: () => authGet('/admin/geo/status'),
}

const users = {
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/users' + (q ? '?' + q : ''))
  },
  create: (body) => apiPost('/admin/users', body),
  update: (id, body) => apiPut('/admin/users/' + id, body),
  remove: (id) => apiDelete('/admin/users/' + id),
  batchDelete: (ids) => apiPost('/admin/users/batch-delete', { ids }),
  downloads: (id, params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/users/' + id + '/downloads' + (q ? '?' + q : ''))
  },
  // 管理员手动解绑设备：PUT /admin/users/:id/unbind-device（不受 72h 冷却限制）
  unbindDevice: (id) => apiPut('/admin/users/' + id + '/unbind-device', {}),
}

const orders = {
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/orders' + (q ? '?' + q : ''))
  },
  batchDelete: (ids) => apiPost('/admin/orders/batch-delete', { ids }),
}

const coupons = {
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/coupons' + (q ? '?' + q : ''))
  },
  create: (body) => apiPost('/admin/coupons', body),
  update: (id, body) => apiPut('/admin/coupons/' + id, body),
  remove: (id) => apiDelete('/admin/coupons/' + id),
  generate: (body) => apiPost('/admin/coupons/generate', body),
  batchDelete: (ids) => apiPost('/admin/coupons/batch-delete', { ids }),
}

const navigations = {
  list: (position = 'top') => authGet('/admin/navigations?position=' + position),
  // 后端契约：自定义项 { isCustom:true, status }；分类项 { isCustom:false, showInTop | showInMobile }
  toggle: (id, payload) => apiPut('/admin/navigations/' + id + '/toggle', payload),
  batchSort: (items) => apiPut('/admin/navigations/batch-sort', { items }),
  createCustom: (body) => apiPost('/admin/navigations/custom', body),
  updateCustom: (id, body) => apiPut('/admin/navigations/custom/' + id, body),
  removeCustom: (id) => apiDelete('/admin/navigations/custom/' + id),
  sync: () => apiPost('/admin/navigations/sync-categories', {}),
}

const settings = {
  all: () => authGet('/admin/settings'),
  update: (body) => apiPut('/admin/settings', body),
  updateSite: (body) => apiPut('/admin/settings/site', body),
  updateRegister: (body) => apiPut('/admin/settings/register', body),
  updateEmail: (body) => apiPut('/admin/settings/email', body),
  updatePayment: (body) => apiPut('/admin/settings/payment', body),
  updateStorage: (body) => apiPut('/admin/settings/storage', body),
  // R2 存量文件迁移：触发 + 进度查询
  r2Migrate: (includeInstaller) =>
    authPost('/admin/r2/migrate' + (includeInstaller ? '?includeInstaller=true' : ''), {}),
  r2MigrateStatus: () => authGet('/admin/r2/migrate/status'),
  // 首页 Banner（电脑首页 / 手机首页）：按 device 读取 / 保存
  getHomeBanner: (device) => authGet('/admin/settings/home-banner?device=' + device),
  updateHomeBanner: (body) => apiPut('/admin/settings/home-banner', body),
  // 首页分类目录（电脑首页 / 手机首页）：读取走公开接口（仅有序分类 ID，无需鉴权）/ 保存走管理接口
  getHomeCategories: (device) => apiGet('/home-categories?device=' + device),
  updateHomeCategories: (body) => apiPut('/admin/settings/home-categories', body),
  // 首页展示配置（卡片风格 / 每行几个，按设备）：读取走管理接口
  getHomeDisplay: (device) => authGet('/admin/settings/home-display?device=' + device),
  updateHomeDisplay: (body) => apiPut('/admin/settings/home-display', body),
  // 上传首页 Banner 文件（图片 / 视频），需鉴权 + FormData
  uploadBanner: async (file) => {
    const base =
      typeof window === 'undefined'
        ? (process.env.API_BASE_INTERNAL || 'http://localhost:8080/api')
        : (process.env.NEXT_PUBLIC_API_BASE || '/api')
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(base + '/admin/upload-banner', {
      method: 'POST',
      headers: authHeaders(),
      body: form,
    })
    if (!res.ok) {
      let data = {}
      try { data = await res.json() } catch (e) { /* ignore */ }
      const err = new Error('上传失败 ' + res.status)
      err.status = res.status
      err.data = data
      throw err
    }
    return res.json()
  },
  // 上传站点 Logo（图片 / SVG），需鉴权 + FormData
  uploadLogo: async (file) => {
    const base =
      typeof window === 'undefined'
        ? (process.env.API_BASE_INTERNAL || 'http://localhost:8080/api')
        : (process.env.NEXT_PUBLIC_API_BASE || '/api')
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(base + '/admin/upload-logo', {
      method: 'POST',
      headers: authHeaders(),
      body: form,
    })
    if (!res.ok) {
      let data = {}
      try { data = await res.json() } catch (e) { /* ignore */ }
      const err = new Error('上传失败 ' + res.status)
      err.status = res.status
      err.data = data
      throw err
    }
    return res.json()
  },
  // 上传客户端安装包（exe/msi/dmg/zip/appimage），需鉴权 + FormData
  uploadInstaller: async (file) => {
    const base =
      typeof window === 'undefined'
        ? (process.env.API_BASE_INTERNAL || 'http://localhost:8080/api')
        : (process.env.NEXT_PUBLIC_API_BASE || '/api')
    const form = new FormData()
    form.append('file', file)
    const res = await fetch(base + '/admin/upload-installer', {
      method: 'POST',
      headers: authHeaders(),
      body: form,
    })
    if (!res.ok) {
      let data = {}
      try { data = await res.json() } catch (e) { /* ignore */ }
      const err = new Error('上传失败 ' + res.status)
      err.status = res.status
      err.data = data
      throw err
    }
    return res.json()
  },
  // 上传会员体系图标（组图标 / 等级图标，支持 ico/png/jpg/jpeg/gif/webp/svg）
  uploadMembershipIcon,
}

// 客户端配置（Electron 桌面端）：整体 JSON 读写
const clientConfig = {
  get: () => authGet('/admin/client-config'),
  // 后端契约要求 { params: {...} }（AdminUpdateClientConfig 只认 json:"params"）；
  // 旧后台曾直接发裸对象导致 config 被写成字面量 "null"，此处统一包一层。
  update: (body) => apiPut('/admin/client-config', { params: body }),
  // 后端代拉 GitHub 最新 Release（避免 admin 端直连 api.github.com 不可达）
  githubLatest: () => authGet('/admin/github-release/latest'),
  // 后台把安装包发布到 GitHub Releases（multipart，复用全局 github_token）
  githubUpload: (formData) => apiPostForm('/admin/github-release/upload', formData),
}

// 红包雨配置：仅读写 client_config.redPacket 段，安全隔离其它客户端配置
const redPacket = {
  get: () => authGet('/admin/client-config'),
  update: (body) => apiPost('/admin/redpacket', body),
}

// 安卓端配置（原生 Android App）：整体 JSON 读写，独立于 PC「客户端配置」
const androidConfig = {
  get: () => authGet('/admin/android-config'),
  // 同 clientConfig：后端 AdminUpdateAndroidConfig 只认 json:"params"，统一包 { params }。
  update: (body) => apiPut('/admin/android-config', { params: body }),
}

// 文章附件（下载模块 / 音乐模块的资源）：列表 + 增删改
const attachments = {
  list: (articleId) => authGet('/articles/' + articleId + '/attachments'),
  // 新建走 multipart（后端 CreateAttachment 读取 PostForm，网盘/外链无需上传文件）
  create: (articleId, formData) => apiPostForm('/articles/' + articleId + '/attachments', formData),
  update: (id, body) => apiPut('/attachments/' + id, body),
  remove: (id) => apiDelete('/attachments/' + id),
}

// 视频源（视频模块）：列表 + 增删改
const videos = {
  list: (articleId) => authGet('/articles/' + articleId + '/video-sources'),
  add: (articleId, body) => apiPost('/articles/' + articleId + '/video-sources', body),
  update: (id, body) => apiPut('/video-sources/' + id, body),
  remove: (id) => apiDelete('/video-sources/' + id),
  // 解析 x.com / youtube / bilibili 链接，返回 {title, thumbnail, sourceType}
  parseInfo: (url) => authGet('/video/parse-info?url=' + encodeURIComponent(url)),
}

// 章节（漫画 / 小说模块）：列表(公开) + 增删改(admin)
const chapters = {
  list: (contentId) => authGet('/content/chapters/' + contentId),
  create: (body) => apiPost('/admin/chapters', body),
  update: (id, body) => apiPut('/admin/chapters/' + id, body),
  remove: (id) => apiDelete('/admin/chapters/' + id),
}

// 会员等级配置（还原旧 Vue 后台的会员等级管理）
const membership = {
  list: () => authGet('/admin/membership-levels'),
  create: (body) => apiPost('/admin/membership-levels', body),
  update: (key, body) => apiPut('/admin/membership-levels/' + key, body),
  remove: (key) => apiDelete('/admin/membership-levels/' + key),
  seed: () => apiPost('/admin/membership-levels/seed', {}),
}

// 新会员体系（双组 VIP/SVIP + 经验等级，1.0.5 双端启用）。与旧 membership 并存，
// 旧等级继续服务老客户端；本组接口供后台配置新体系，并供 1.0.5 客户端拉取。
const membershipV2 = {
  list: () => authGet('/admin/membership-v2'),
  updateGroup: (groupKey, body) => apiPut('/admin/membership-v2/groups/' + groupKey, body),
  createTier: (body) => apiPost('/admin/membership-v2/tiers', body),
  updateTier: (id, body) => apiPut('/admin/membership-v2/tiers/' + id, body),
  removeTier: (id) => apiDelete('/admin/membership-v2/tiers/' + id),
  getTaskExp: () => authGet('/admin/membership-v2/task-exp'),
  updateTaskExp: (body) => apiPut('/admin/membership-v2/task-exp', body),
  seed: () => apiPost('/admin/membership-v2/seed', {}),
  // 上传会员图标（组图标 / 等级图标），返回 { url }
  uploadIcon: (file) => uploadMembershipIcon(file),
}

// 百度网盘 Cookie 配置（实时检测失效）
const baidu = {
  get: () => authGet('/admin/baidu-cookie'),
  update: (body) => apiPut('/admin/baidu-cookie', body),
  test: () => apiPost('/admin/baidu-cookie/test', {}),
  status: () => authGet('/admin/baidu-cookie/status'),
  // 前端辅助迁移：解析分享（可传普通百度 cookie 拿真实 fs_id）+ 迁移到系统账号 + 查询迁移记录
  resolve: (body) => apiPost('/admin/baidu/resolve', body),
  migrate: (body) => apiPost('/admin/baidu/migrate', body),
  migrations: (articleId) => authGet('/admin/baidu/migrations?articleId=' + articleId),
}

// SEO 管理
const seo = {
  settings: () => authGet('/admin/seo/settings'),
  updateSettings: (body) => apiPut('/admin/seo/settings', body),
  push: (engines) => apiPost('/admin/seo/push', { engines }),
  pseudoOriginal: (articleId, prompt) => apiPost('/admin/seo/pseudo-original', { articleId, prompt }),
  insertKeywords: (articleId, keywords) => apiPost('/admin/seo/insert-keywords', { articleId, keywords }),
  cleanKeywords: (articleId) => apiPost('/admin/seo/clean-keywords', { articleId }),
  insertKeywordsAll: (keywords) => apiPost('/admin/seo/insert-keywords-all', { keywords }),
  cleanKeywordsAll: () => apiPost('/admin/seo/clean-keywords-all'),
}

// 爬虫访问监控
const crawler = {
  summary: (date) => authGet('/admin/crawler-logs/summary' + (date ? '?date=' + date : '')),
  logs: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/crawler-logs' + (q ? '?' + q : ''))
  },
}

// 邮件群发（每天 400 封、去重、最后一封给 gmshe@qq.com）
const email = {
  status: () => authGet('/admin/email/status'),
  importFile: (formData) => apiPostForm('/admin/email/import', formData),
  importPath: (filePath) => apiPost('/admin/email/import', { filePath }),
  importEmails: (emails) => apiPost('/admin/email/import', { emails }),
  sendBatch: (limit) => apiPost('/admin/email/send-batch', limit ? { limit } : {}),
  reset: () => apiPost('/admin/email/reset', {}),
}

// 站内通知群发（所有用户 / 免费注册用户 / 各等级会员）
const notifications = {
  // 各投递对象人数统计
  targets: () => authGet('/admin/notifications/targets'),
  // 群发历史
  list: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/notifications' + (q ? '?' + q : ''))
  },
  // 发送：{ title, content, target: 'all'|'free'|'levels', levels: [], attachments: [{name,url,size}] }
  send: (body) => apiPost('/admin/notifications', body),
  remove: (id) => apiDelete('/admin/notifications/' + id),
  recall: (id) => apiPost('/admin/notifications/' + id + '/recall', {}),
  // 图片 / 附件上传（复用后台通用上传，自动走 R2）
  upload: (formData) => apiPostForm('/admin/upload-generic', formData),
}

// 客服聊天（用户 ↔ 管理员 即时通讯）：会话列表 / 消息记录 / 管理员回复
const chat = {
  // 会话列表：与客服管理员聊过天的用户（排除管理员自身）
  conversations: (params = {}) => {
    const q = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/chat/conversations' + (q ? '?' + q : ''))
  },
  // 与某用户的聊天记录：{ list, total }
  messages: (userId, params = {}) => {
    const q = new URLSearchParams(
      Object.entries({ userId, ...params }).filter(([, v]) => v !== '' && v != null)
    ).toString()
    return authGet('/admin/chat/messages' + (q ? '?' + q : ''))
  },
  // 管理员回复用户：{ userId, content, email }
  reply: (body) => apiPost('/admin/chat/reply', body),

  // ===== 客服自动回复规则（前端 App「联系客服」自动应答）=====
  // 规则列表（含禁用）：返回 ChatAutoReply 数组
  autoReplies: () => authGet('/admin/chat/auto-replies'),
  // 新建规则：{ name, matchType, keyword, replyType, replyContent, priority }
  createAutoReply: (body) => apiPost('/admin/chat/auto-replies', body),
  // 更新规则：同上字段（enabled 可单独切开关）
  updateAutoReply: (id, body) => apiPut('/admin/chat/auto-replies/' + id, body),
  // 删除规则
  removeAutoReply: (id) => apiDelete('/admin/chat/auto-replies/' + id),
  // 开场问候语：{ greeting }
  greeting: () => authGet('/admin/chat/greeting'),
  // 更新开场问候语：{ greeting }
  updateGreeting: (body) => apiPut('/admin/chat/greeting', body),
}

// 工具箱：文章导入（测试一篇 / 定时每日发布）+ 数据/文件清理
const tools = {
  // 测试导入一篇（limit=1）：formData 需含 file
  importExcelTest: (formData) => apiPostForm('/admin/articles/import-excel', formData),
  // 定时导入（固定/随机每天 N 篇）：formData 需含 file 及 mode/perDay/time/startDate/categoryId/releaseStatus
  importScheduled: (formData) => apiPostForm('/admin/articles/import-scheduled', formData),
  // 数据库冗余清理（dry-run 或 execute）
  dbCleanup: (body) => apiPost('/admin/tools/db-cleanup', body),
  // 网站旧文件 / 缓存清理（dry-run 预览大小 或 execute）
  filesCleanup: (body) => apiPost('/admin/tools/files-cleanup', body),
  // 定时发布列表
  scheduledList: () => authGet('/admin/scheduled/list'),
  // 取消定时发布（删除未发布草稿）
  scheduledCancel: (ids) => apiPost('/admin/scheduled/cancel', { ids }),
}

// API 资源采集（MacCMS v10 XML）：采集接口、解析/播放接口、分类映射、手动采集、清空
const collector = {
  // 运行状态 + 配置摘要 + 已采集数量 + 分类映射
  status: () => authGet('/admin/collect/status'),
  // 读取完整配置（用于表单回填）
  config: () => authGet('/admin/collect/config'),
  // 保存配置：{ collect_api_url, collect_player_parse_url, collect_enabled, collect_category_id, collect_category_map(JSON字符串), collect_pages, collect_hour, collect_max_per_run, collect_update_existing }
  saveConfig: (body) => apiPost('/admin/collect/config', body),
  // 拉取采集源自带分类列表（手动映射用）
  sourceCategories: () => authGet('/admin/collect/source-categories'),
  // 站点分类列表（映射目标用）
  siteCategories: () => authGet('/admin/categories'),
  // 立即采集：{ pages, startPage, categoryId }（有映射时按映射逐类）
  run: (body) => apiPost('/admin/collect/run', body),
  // 清空所有采集内容（external_ref 非空）及其视频源
  clear: () => apiPost('/admin/collect/clear', {}),
}

// 积分体系管理（行为发分配置在 daily_tasks；本模块管：积分抵扣下载开关、注册赠送、调分、积分商城）
const points = {
  config: () => authGet('/admin/points/config'),
  updateConfig: (body) => apiPut('/admin/points/config', body),
  adjust: (body) => apiPost('/admin/points/adjust', body),
  // 积分商城商品复用既有 /admin/points-products 接口
  products: () => authGet('/admin/points-products'),
  createProduct: (body) => apiPost('/admin/points-products', body),
  updateProduct: (id, body) => apiPut('/admin/points-products/' + id, body),
  deleteProduct: (id) => apiDelete('/admin/points-products/' + id),
  // 行为积分策略（每日任务发分配置）
  tasks: () => authGet('/admin/points/tasks'),
  updateTask: (id, body) => apiPut('/admin/points/tasks/' + id, body),
}

// 多端统一配置（按端读写参数；功能开关 / 策略独立管理）
const multiConfig = {
  get: (clientType) => authGet('/admin/multi-config/' + clientType),
  update: (clientType, params) => apiPut('/admin/multi-config/' + clientType, { params }),
}

// 功能开关（FeatureFlag）
const featureFlags = {
  list: (clientType) =>
    authGet('/admin/feature-flags' + (clientType ? '?clientType=' + clientType : '')),
  create: (body) => apiPost('/admin/feature-flags', body),
  update: (id, body) => apiPut('/admin/feature-flags/' + id, body),
  remove: (id) => apiDelete('/admin/feature-flags/' + id),
}

// 策略（ClientStrategy：强制更新 / 红包时间窗 / PCDN 等）
const strategies = {
  list: (clientType) =>
    authGet('/admin/strategies' + (clientType ? '?clientType=' + clientType : '')),
  create: (body) => apiPost('/admin/strategies', body),
  update: (id, body) => apiPut('/admin/strategies/' + id, body),
  remove: (id) => apiDelete('/admin/strategies/' + id),
}

// 网站端配置（合并逻辑，兼容旧端）
const webConfig = {
  get: () => authGet('/admin/web-config'),
  update: (params) => apiPut('/admin/web-config', { params }),
}

// 在线客服：坐席 / 队列 / 监控 / 路由 / 会话控制
const cs = {
  agents: () => authGet('/admin/cs/agents'),
  createAgent: (body) => apiPost('/admin/cs/agents', body),
  updateAgent: (id, body) => apiPut('/admin/cs/agents/' + id, body),
  removeAgent: (id) => apiDelete('/admin/cs/agents/' + id),
  queue: () => authGet('/admin/cs/queue'),
  monitor: () => authGet('/admin/cs/monitor'),
  assign: (convId, agentId) => apiPost('/admin/cs/conversations/' + convId + '/assign', { agentId }),
  takeover: (convId) => apiPost('/admin/cs/conversations/' + convId + '/takeover', {}),
  transfer: (convId, agentId, reason) =>
    apiPost('/admin/cs/conversations/' + convId + '/transfer', { agentId, reason }),
  close: (convId, summary) => apiPost('/admin/cs/conversations/' + convId + '/close', { summary }),
  routingRules: () => authGet('/admin/cs/routing-rules'),
  createRoutingRule: (body) => apiPost('/admin/cs/routing-rules', body),
  updateRoutingRule: (id, body) => apiPut('/admin/cs/routing-rules/' + id, body),
  removeRoutingRule: (id) => apiDelete('/admin/cs/routing-rules/' + id),
  updateRouting: (strategy) => apiPut('/admin/cs/routing', { strategy }),
  getRouting: () => authGet('/admin/cs/routing'),
}

// 周签到计划（checkin_plan）：数组，长度=一轮天数
const checkinPlan = {
  get: () => authGet('/admin/checkin-plan'),
  update: (plan) => apiPut('/admin/checkin-plan', plan),
}

// 带 Bearer 的 CSV 导出下载（返回 blob URL）
export async function downloadCouponsCsv(params = {}) {
  const base = (typeof window === 'undefined'
    ? (process.env.API_BASE_INTERNAL || 'http://localhost:8080/api')
    : (process.env.NEXT_PUBLIC_API_BASE || '/api'))
  const q = new URLSearchParams(
    Object.entries(params).filter(([, v]) => v !== '' && v != null)
  ).toString()
  const res = await fetch(base + '/admin/coupons/export' + (q ? '?' + q : ''), {
    headers: { Authorization: 'Bearer ' + (getToken() || '') },
  })
  if (!res.ok) throw new Error('导出失败 ' + res.status)
  const blob = await res.blob()
  return URL.createObjectURL(blob)
}

export const adminApi = {
  dashboard,
  articles,
  categories,
  pages,
  comments,
  tags,
  geo,
  users,
  orders,
  coupons,
  navigations,
  settings,
  clientConfig,
  redPacket,
  androidConfig,
  membership,
  membershipV2,
  attachments,
  videos,
  chapters,
  seo,
  crawler,
  baidu,
  email,
  notifications,
  chat,
  tools,
  collector,
  points,
  multiConfig,
  featureFlags,
  strategies,
  webConfig,
  cs,
  checkinPlan,
  downloadCouponsCsv,
}
