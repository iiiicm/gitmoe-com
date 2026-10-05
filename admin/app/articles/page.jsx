'use client'

import { useEffect, useState, useCallback, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { adminApi } from '../../lib/adminApi'

// 六大文章模块：新闻(普通文章,默认) / 下载 / 视频 / 漫画 / 小说 / 音乐(纯免费展示)
const CONTENT_TYPES = [
  { v: 'article', label: '新闻模块', desc: '普通文章' },
  { v: 'download', label: '下载模块', desc: '付费下载' },
  { v: 'video', label: '视频模块', desc: '视频解析' },
  { v: 'comic', label: '漫画模块', desc: '漫画阅读' },
  { v: 'novel', label: '小说模块', desc: '小说阅读' },
  { v: 'music', label: '音乐模块', desc: '免费展示' },
]
// 列表筛选 / 批量改模块：音乐模块已暂时下线，不在此出现（存量音乐文章在表格中仍按其标签显示）。
const VISIBLE_CONTENT_TYPES = CONTENT_TYPES.filter((t) => t.v !== 'music')

const STATUS_OPTS = [
  { v: 'published', label: '已发布' },
  { v: 'pending', label: '待审核' },
  { v: 'draft', label: '草稿' },
]

// 游戏类型（与前端分类页 GENRE_ORDER 保持一致）：SLG / RPG / ADV / ACT / HTML / 其他
const GENRE_ORDER = ['SLG', 'RPG', 'ADV', 'ACT', 'HTML', '其他']

function AdminArticlesInner() {
  const router = useRouter()
  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const [cats, setCats] = useState([])
  const [catMap, setCatMap] = useState({})
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(15)
  const [filters, setFilters] = useState({ keyword: '', contentType: '', status: '', categoryId: '', broken: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [selected, setSelected] = useState([])
  const [batchCat, setBatchCat] = useState('') // 批量改分类的目标分类
  const [batchStatus, setBatchStatus] = useState('') // 批量改状态的目标状态
  const [batchModule, setBatchModule] = useState('') // 批量改模块的目标内容类型
  const [batchGenre, setBatchGenre] = useState('') // 批量改游戏类型的目标类型
  const [priceModal, setPriceModal] = useState(false) // 批量改价格弹窗
  const [priceDraft, setPriceDraft] = useState({ allPrice: 0, registeredPrice: '', memberPrices: {} })
  const [msg, setMsg] = useState('')
  const [levels, setLevels] = useState([]) // 会员等级（不含 free）

  // F8：侧边栏「新建」= /articles?action=create → 跳转整页新建编辑器。
  const searchParams = useSearchParams()
  const actionParam = searchParams.get('action')
  useEffect(() => {
    if (actionParam === 'create') router.push('/articles/new')
  }, [actionParam, router])

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await adminApi.articles.list({ ...filters, page, pageSize })
      if (res.code === 0) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        setError(res.message || '加载失败')
      }
    } catch (e) {
      setError(e?.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [filters, page, pageSize])

  useEffect(() => {
    ;(async () => {
      try {
        const c = await adminApi.categories.list()
        if (c.code === 0) {
          setCats(c.data || [])
          const m = {}
          ;(c.data || []).forEach((x) => { m[x.id] = x })
          setCatMap(m)
        }
      } catch (e) { /* ignore */ }
    })()
  }, [])

  // 加载会员等级，用于各会员价输入
  useEffect(() => {
    ;(async () => {
      try {
        const m = await adminApi.membership.list()
        if (m.code === 0) {
          setLevels((m.data || []).filter((l) => (l.key || '').toLowerCase() !== 'free'))
        }
      } catch (e) { /* ignore */ }
    })()
  }, [])

  useEffect(() => { load() }, [load])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const toggleSelect = (id) =>
    setSelected((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))
  const allChecked = rows.length > 0 && rows.every((r) => selected.includes(r.id))
  const toggleAll = () =>
    setSelected(allChecked ? [] : rows.map((r) => r.id))

  // 新建 / 编辑 → 跳转整页编辑器（不再用弹窗）。
  const openCreate = () => router.push('/articles/new')
  const openEdit = (row) => router.push(`/articles/${row.id}/edit`)

  const remove = async (id) => {
    if (!confirm('确认删除该文章？')) return
    try {
      const res = await adminApi.articles.remove(id)
      if (res.code === 0) { setMsg('已删除'); load() }
      else setError(res.message || '删除失败')
    } catch (e) { setError(e?.message || '删除失败') }
  }

  const batchDelete = async () => {
    if (selected.length === 0) return
    if (!confirm(`确认删除选中的 ${selected.length} 篇文章？`)) return
    try {
      const res = await adminApi.articles.batchDelete(selected)
      if (res.code === 0) { setMsg(`已删除 ${selected.length} 篇`); setSelected([]); load() }
      else setError(res.message || '批量删除失败')
    } catch (e) { setError(e?.message || '批量删除失败') }
  }

  const batchChangeCat = async () => {
    if (selected.length === 0 || !batchCat) return
    const catName = catMap[batchCat]?.name || ('#' + batchCat)
    if (!confirm(`确认将选中的 ${selected.length} 篇文章移动到分类「${catName}」？`)) return
    try {
      const res = await adminApi.articles.batchChangeCategory(selected, Number(batchCat))
      if (res.code === 0) { setMsg('已批量修改分类'); setSelected([]); setBatchCat(''); load() }
      else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const batchChangeStatus = async () => {
    if (selected.length === 0 || !batchStatus) return
    const label = STATUS_OPTS.find((t) => t.v === batchStatus)?.label || batchStatus
    if (!confirm(`确认将选中的 ${selected.length} 篇文章状态改为「${label}」？`)) return
    try {
      const res = await adminApi.articles.batchChangeStatus(selected, batchStatus)
      if (res.code === 0) { setMsg('已批量修改状态'); setSelected([]); setBatchStatus(''); load() }
      else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const batchChangeModule = async () => {
    if (selected.length === 0 || !batchModule) return
    const label = CONTENT_TYPES.find((t) => t.v === batchModule)?.label || batchModule
    if (!confirm(`确认将选中的 ${selected.length} 篇文章模块改为「${label}」？`)) return
    try {
      const res = await adminApi.articles.batchChangeModule(selected, batchModule)
      if (res.code === 0) { setMsg('已批量修改模块'); setSelected([]); setBatchModule(''); load() }
      else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const batchChangeGenre = async () => {
    if (selected.length === 0 || !batchGenre) return
    if (!confirm(`确认将选中的 ${selected.length} 篇文章游戏类型设为「${batchGenre}」？`)) return
    try {
      const res = await adminApi.articles.batchChangeGenre(selected, batchGenre)
      if (res.code === 0) { setMsg(`已批量设置类型（${res.data?.updated || 0} 篇）`); setSelected([]); setBatchGenre(''); load() }
      else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const batchAutoMatch = async () => {
    if (selected.length === 0) return
    if (!confirm(`确认根据标题自动识别 ${selected.length} 篇文章的游戏设备/类型？（仅对未设置类型的下载类文章生效）`)) return
    try {
      const res = await adminApi.articles.batchAutoMatchGenre(selected, false)
      if (res.code === 0) { setMsg(`自动匹配完成：扫描 ${res.data?.scanned || 0} 篇，匹配 ${res.data?.matched || 0} 篇`); setSelected([]); load() }
      else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const openPriceModal = () => {
    if (selected.length === 0) return
    setPriceDraft({ allPrice: 0, registeredPrice: '', memberPrices: {} })
    setPriceModal(true)
  }

  const setMemberPrice = (key, val) =>
    setPriceDraft((d) => ({ ...d, memberPrices: { ...(d.memberPrices || {}), [key]: val } }))

  const submitBatchPrice = async () => {
    setMsg('')
    try {
      const payload = {
        allPrice: Number(priceDraft.allPrice) || 0,
      }
      if (priceDraft.registeredPrice !== '' && priceDraft.registeredPrice != null) {
        payload.registeredPrice = Number(priceDraft.registeredPrice) || 0
      }
      const mp = {}
      levels.forEach((l) => {
        const v = priceDraft.memberPrices?.[l.key]
        if (v !== '' && v != null && Number(v) > 0) mp[l.key] = Number(v)
      })
      payload.memberPrices = mp
      const res = await adminApi.articles.batchSetPrice(selected, payload)
      if (res.code === 0) {
        setMsg(`已批量设置价格（${res.data?.updated || 0} 篇）`)
        setPriceModal(false); setSelected([]); load()
      } else setError(res.message || '操作失败')
    } catch (e) { setError(e?.message || '操作失败') }
  }

  const moduleLabel = (v) => CONTENT_TYPES.find((t) => t.v === v)?.label || v

  return (
    <div>
      <div className="admin-section-title"><i className="fa-solid fa-file-lines" /> 文章管理</div>
      <div className="admin-hint">每篇文章需选择对应模块；不选则默认「新闻模块」（普通文章）。下载/视频/漫画/小说为付费模块。点击「新建文章」或列表中的「编辑」会打开整页编辑器（标题/正文/摘要 + 右侧元框），模块专属资源（度盘/视频/章节）在编辑器内的对应元框中维护。音乐模块已暂时下线。</div>
      {error && <div className="admin-error">{error}</div>}
      {msg && <div className="admin-success">{msg}</div>}

      <div className="admin-toolbar">
        <input className="admin-input" placeholder="搜索标题…"
          value={filters.keyword}
          onChange={(e) => setFilters((f) => ({ ...f, keyword: e.target.value }))}
          onKeyDown={(e) => e.key === 'Enter' && (setPage(1), load())} />
        <select className="admin-select" value={filters.contentType}
          onChange={(e) => setFilters((f) => ({ ...f, contentType: e.target.value }))}>
          <option value="">全部模块</option>
          {VISIBLE_CONTENT_TYPES.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
        </select>
        <select className="admin-select" value={filters.status}
          onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}>
          <option value="">全部状态</option>
          {STATUS_OPTS.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
        </select>
        <select className="admin-select" value={filters.categoryId}
          onChange={(e) => setFilters((f) => ({ ...f, categoryId: e.target.value }))}>
          <option value="">全部分类</option>
          {cats.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select className="admin-select" value={filters.broken}
          onChange={(e) => setFilters((f) => ({ ...f, broken: e.target.value }))}>
          <option value="">全部图片</option>
          <option value="all">仅失效图片</option>
          <option value="cover">仅失效封面</option>
          <option value="content">仅正文失效图</option>
        </select>
        <button className="admin-btn ghost" onClick={() => { setPage(1); load() }}>查询</button>
        <div className="spacer" />
        {selected.length > 0 && (
          <span className="batch-bar">
            已选 {selected.length} 篇：
            <select className="admin-select sm" value={batchModule}
              onChange={(e) => setBatchModule(e.target.value)}>
              <option value="">更改模块…</option>
              {VISIBLE_CONTENT_TYPES.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
            </select>
            <button className="admin-btn ghost sm" onClick={batchChangeModule} disabled={!batchModule}>改模块</button>
            <select className="admin-select sm" value={batchCat}
              onChange={(e) => setBatchCat(e.target.value)}>
              <option value="">移动至分类…</option>
              {cats.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
            <button className="admin-btn ghost sm" onClick={batchChangeCat} disabled={!batchCat}>改分类</button>
            <select className="admin-select sm" value={batchStatus}
              onChange={(e) => setBatchStatus(e.target.value)}>
              <option value="">更改状态…</option>
              {STATUS_OPTS.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
            </select>
            <button className="admin-btn ghost sm" onClick={batchChangeStatus} disabled={!batchStatus}>改状态</button>
            <select className="admin-select sm" value={batchGenre}
              onChange={(e) => setBatchGenre(e.target.value)}>
              <option value="">设置类型…</option>
              {GENRE_ORDER.map((g) => <option key={g} value={g}>{g}</option>)}
            </select>
            <button className="admin-btn ghost sm" onClick={batchChangeGenre} disabled={!batchGenre}>改类型</button>
            <button className="admin-btn ghost sm" onClick={batchAutoMatch}><i className="fa-solid fa-wand-magic-sparkles" /> 自动匹配</button>
            <button className="admin-btn ghost sm" onClick={openPriceModal}><i className="fa-solid fa-tags" /> 批量改价</button>
          </span>
        )}
        <button className="admin-btn danger sm" onClick={batchDelete} disabled={selected.length === 0}>
          批量删除
        </button>
        <button className="admin-btn" onClick={openCreate}><i className="fa-solid fa-plus" /> 新建文章</button>
      </div>

      <div className="admin-table-wrap">
        <table className="admin-table">
          <thead>
            <tr>
              <th style={{ width: 36 }}>
                <input type="checkbox" checked={allChecked} onChange={toggleAll} />
              </th>
              <th style={{ width: 64 }}>ID</th>
              <th>封面</th>
              <th>标题</th>
              <th>模块</th>
              <th>分类</th>
              <th>状态</th>
              <th>浏览</th>
              <th>发布时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr><td colSpan={10}><div className="admin-empty">暂无文章</div></td></tr>
            ) : rows.map((r) => (
              <tr key={r.id}>
                <td><input type="checkbox" checked={selected.includes(r.id)} onChange={() => toggleSelect(r.id)} /></td>
                <td className="admin-muted">{r.id}</td>
                <td>{r.cover ? <img className="thumb-sm" src={r.cover} alt="" /> : <div className="thumb-sm" />}</td>
                <td style={{ maxWidth: 260 }}>{r.title || <span className="admin-muted">（无标题）</span>}</td>
                <td>
                  <span className="admin-badge blue">
                    {moduleLabel(r.contentType)}
                  </span>
                </td>
                <td>{r.categoryId != null && catMap[r.categoryId] ? catMap[r.categoryId].name : <span className="admin-muted">—</span>}</td>
                <td>
                  <span className={'admin-badge ' + (r.status === 'published' ? 'green' : r.status === 'pending' ? 'orange' : 'gray')}>
                    {STATUS_OPTS.find((t) => t.v === r.status)?.label || r.status}
                  </span>
                </td>
                <td>{r.views ?? 0}</td>
                <td className="admin-muted">{(r.publishedAt || r.createdAt || '').slice(0, 10)}</td>
                <td>
                  <div className="row-actions">
                    <button className="link-btn" onClick={() => openEdit(r)}>编辑</button>
                    <button className="link-btn danger" onClick={() => remove(r.id)}>删除</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="admin-pagination">
        <span className="page-size">
          每页
          <select className="admin-select sm" value={pageSize}
            onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1) }}>
            <option value={10}>10</option>
            <option value={15}>15</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
          篇
        </span>
        <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>上一页</button>
        <span className="page-info">第 {page} / {totalPages} 页 · 共 {total} 条</span>
        <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>下一页</button>
      </div>

      {/* ===== 批量改价格弹窗 ===== */}
      {priceModal && (
        <div className="admin-modal-mask" onClick={() => setPriceModal(false)}>
          <div className="admin-modal" onClick={(e) => e.stopPropagation()}>
            <div className="admin-modal-head">
              <h3>批量设置价格（{selected.length} 篇）</h3>
              <button className="admin-modal-close" onClick={() => setPriceModal(false)}>×</button>
            </div>
            <div className="admin-modal-body">
              <div className="admin-hint">
                所有用户价格 = 基础价（免费用户及未单独设置会员价的等级均按此收费）；注册用户价格写入「免费会员」档，可区别于基础价；各会员价仅填写需要单独设置的等级，留空则不修改该等级（保持 0 / 免费）。
              </div>
              <div className="admin-form-row">
                <div className="admin-field">
                  <label>所有用户价格 (¥)</label>
                  <input className="admin-input" type="number" min="0" step="0.01" style={{ width: '100%' }}
                    value={priceDraft.allPrice}
                    onChange={(e) => setPriceDraft((d) => ({ ...d, allPrice: e.target.value }))} />
                </div>
                <div className="admin-field">
                  <label>注册用户价格 (¥，免费会员)</label>
                  <input className="admin-input" type="number" min="0" step="0.01" style={{ width: '100%' }}
                    value={priceDraft.registeredPrice}
                    onChange={(e) => setPriceDraft((d) => ({ ...d, registeredPrice: e.target.value }))}
                    placeholder="留空=与基础价相同" />
                </div>
              </div>
              <div className="admin-subtitle">各会员价格（留空=不修改，按基础价或会员折扣）</div>
              <div className="admin-form-row">
                {levels.length === 0 ? (
                  <div className="admin-muted" style={{ fontSize: 12 }}>
                    暂无其他会员等级（可在「会员等级」中配置）。
                  </div>
                ) : levels.map((l) => (
                  <div className="admin-field" key={l.key}>
                    <label>{l.name} 价 (¥)</label>
                    <input className="admin-input" type="number" min="0" step="0.01" style={{ width: '100%' }}
                      value={priceDraft.memberPrices?.[l.key] ?? ''}
                      onChange={(e) => setMemberPrice(l.key, e.target.value)}
                      placeholder="留空不修改" />
                  </div>
                ))}
              </div>
            </div>
            <div className="admin-modal-foot">
              <button className="admin-btn ghost" onClick={() => setPriceModal(false)}>取消</button>
              <button className="admin-btn" onClick={submitBatchPrice}>应用价格</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// next/navigation 的 useSearchParams 需要 Suspense 边界（与 app/login 同款处理）。
export default function AdminArticles() {
  return (
    <Suspense fallback={<div className="admin-loading"><i className="fa-solid fa-spinner fa-spin" /> 加载中…</div>}>
      <AdminArticlesInner />
    </Suspense>
  )
}
