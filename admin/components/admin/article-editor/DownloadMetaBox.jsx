'use client'

import { useEffect, useState, useCallback, useRef } from 'react'
import { adminApi } from '../../../lib/adminApi'
import Metabox from './Metabox'
import { Field, TextInput, TextArea, Select, Switch, DateInput } from './fields'
import { LINK_TYPES, PLATFORMS, PLATFORM_ORDER, PLATFORM_LABELS, LANGUAGES } from './constants'

// ============ 下载模块编辑器：度盘链接 + 提取码（从 app/articles/page.jsx 原样迁出，内部逻辑一行未改）============
function DownloadEditor({ articleId, ensureArticleId }) {
  const [effectiveId, setEffectiveId] = useState(articleId)
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [draft, setDraft] = useState(null)
  const [msg, setMsg] = useState('')

  useEffect(() => { setEffectiveId(articleId) }, [articleId])

  const ensureId = useCallback(async () => {
    if (effectiveId) return effectiveId
    const id = await ensureArticleId()
    setEffectiveId(id)
    return id
  }, [effectiveId, ensureArticleId])

  const load = useCallback(async () => {
    if (!effectiveId) return
    setLoading(true)
    try {
      const res = await adminApi.attachments.list(effectiveId)
      if (res.code === 0) setItems((res.data || []).filter((x) => x.linkType !== 'music'))
    } catch (e) { setMsg(e?.data?.message || e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [effectiveId])
  useEffect(() => { load() }, [load])

  const [migTarget, setMigTarget] = useState(null) // 百度网盘「解析并迁移」抽屉
  const newDraft = () => setDraft({ id: null, title: '', linkType: 'netdisk', linkPlatform: 'baidu', linkUrl: '', linkCode: '', linkUnzip: '', price: 0, isMemberFree: false })
  const editDraft = (it) => setDraft({ id: it.id, title: it.title, linkType: it.linkType || 'netdisk', linkPlatform: it.linkPlatform || 'baidu', linkUrl: it.linkUrl || '', linkCode: it.linkCode || '', linkUnzip: it.linkUnzip || '', price: it.price || 0, isMemberFree: !!it.isMemberFree })

  const saveDraft = async () => {
    setMsg('')
    if (!draft.title.trim()) { setMsg('请填写资源名称'); return }
    if (!draft.linkUrl.trim()) { setMsg('请填写度盘/下载链接'); return }
    try {
      if (draft.id) {
        const res = await adminApi.attachments.update(draft.id, {
          title: draft.title, linkUrl: draft.linkUrl, linkCode: draft.linkCode, linkUnzip: draft.linkUnzip,
          price: Number(draft.price) || 0, isMemberFree: !!draft.isMemberFree,
        })
        if (res.code !== 0) throw new Error(res.message || '更新失败')
      } else {
        const fd = new FormData()
        fd.append('title', draft.title)
        fd.append('linkType', draft.linkType)
        fd.append('linkPlatform', draft.linkPlatform)
        fd.append('linkUrl', draft.linkUrl)
        fd.append('linkCode', draft.linkCode)
        fd.append('linkUnzip', draft.linkUnzip)
        fd.append('price', Number(draft.price) || 0)
        fd.append('isMemberFree', draft.isMemberFree ? 'true' : 'false')
        const res = await adminApi.attachments.create(effectiveId, fd)
        if (res.code !== 0) throw new Error(res.message || '添加失败')
      }
      setDraft(null); load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '保存失败') }
  }
  const del = async (id) => {
    if (!confirm('确认删除该下载资源？')) return
    const res = await adminApi.attachments.remove(id)
    if (res.code === 0) load(); else setMsg(res.message || '删除失败')
  }

  const uploadLocalFile = async (file, id) => {
    if (!file) return
    setMsg('')
    try {
      const fd = new FormData()
      fd.append('title', file.name)
      fd.append('linkType', 'file')
      fd.append('fileType', file.name.split('.').pop() || 'other')
      fd.append('price', '0')
      fd.append('isMemberFree', 'false')
      fd.append('file', file)
      const res = await adminApi.attachments.create(id, fd)
      if (res.code !== 0) throw new Error(res.message || '上传失败')
      load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '上传失败') }
  }

  return (
    <div className="module-editor">
      {!effectiveId && (
        <div className="admin-hint admin-hint-locked">
          <i className="fa-solid fa-circle-info" /> 首次添加下载资源将自动创建文章草稿，无需先点「保存」。
        </div>
      )}
      <div className="module-editor-title"><i className="fa-solid fa-download" /> 下载资源（度盘链接 / 外部直链）</div>
      {msg && <div className="admin-error">{msg}</div>}
      {loading && <div className="admin-muted">加载中…</div>}
      <div className="item-list">
        {items.length === 0 && !loading && <div className="admin-muted">暂无下载资源，请在下方添加。</div>}
        {items.map((it) => (
          <div className="item-row" key={it.id}>
            <div className="item-main">
              <div className="item-name">{it.title} <span className="admin-badge gray">{it.linkPlatform || it.linkType}</span></div>
              <div className="admin-muted item-sub">
                {it.linkType === 'file' ? (it.fileUrl || it.fileName) : it.linkUrl}
                {it.linkCode ? ' · 提取码: ' + it.linkCode : ''}
                {it.linkUnzip ? ' · 解压码: ' + it.linkUnzip : ''}
                {Number(it.price) > 0 ? ' · ¥' + it.price : ' · 免费'}
                {it.isMemberFree ? ' · 会员免费' : ''}
              </div>
            </div>
            <div className="row-actions">
              {it.linkType === 'netdisk' && it.linkPlatform === 'baidu' && (
                <button className="link-btn" onClick={() => setMigTarget(it)}>
                  <i className="fa-solid fa-cloud-arrow-up" /> 解析并迁移
                </button>
              )}
              <button className="link-btn" onClick={() => editDraft(it)}>编辑</button>
              <button className="link-btn danger" onClick={() => del(it.id)}>删除</button>
            </div>
          </div>
        ))}
      </div>

      {!draft && (
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <button className="admin-btn ghost sm" onClick={async () => { const id = await ensureId(); if (id) newDraft() }}><i className="fa-solid fa-plus" /> 添加网盘/直链</button>
          <label className="admin-btn ghost sm" style={{ cursor: 'pointer' }}>
            <i className="fa-solid fa-cloud-arrow-up" /> 上传本地文件
            <input type="file" style={{ display: 'none' }} onChange={async (e) => { const id = await ensureId(); if (id) uploadLocalFile(e.target.files[0], id); e.target.value = '' }} />
          </label>
        </div>
      )}
      {draft && (
        <div className="item-form">
          <div className="admin-form-row">
            <div className="admin-field" style={{ flex: 2 }}>
              <label>资源名称</label>
              <input className="admin-input" style={{ width: '100%' }} value={draft.title}
                onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))} placeholder="如：汉化版完整包 / 破解补丁" />
            </div>
            <div className="admin-field">
              <label>链接类型</label>
              <select className="admin-select" style={{ width: '100%' }} value={draft.linkType}
                onChange={(e) => setDraft((d) => ({ ...d, linkType: e.target.value }))}>
                {LINK_TYPES.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
              </select>
            </div>
          </div>
          {draft.linkType === 'netdisk' && (
            <div className="admin-form-row">
              <div className="admin-field">
                <label>网盘平台</label>
                <select className="admin-select" style={{ width: '100%' }} value={draft.linkPlatform}
                  onChange={(e) => setDraft((d) => ({ ...d, linkPlatform: e.target.value }))}>
                  {PLATFORMS.map((p) => <option key={p.v} value={p.v}>{p.label}</option>)}
                </select>
              </div>
              <div className="admin-field">
                <label>提取码</label>
                <input className="admin-input" style={{ width: '100%' }} value={draft.linkCode}
                  onChange={(e) => setDraft((d) => ({ ...d, linkCode: e.target.value }))} placeholder="如 9syr（可空）" />
              </div>
              <div className="admin-field">
                <label>解压码</label>
                <input className="admin-input" style={{ width: '100%' }} value={draft.linkUnzip}
                  onChange={(e) => setDraft((d) => ({ ...d, linkUnzip: e.target.value }))} placeholder="如 1234（可空）" />
              </div>
            </div>
          )}
          <div className="admin-field">
            <label>{draft.linkType === 'netdisk' ? '度盘 / 网盘链接' : '外部直链'}</label>
            <input className="admin-input" style={{ width: '100%' }} value={draft.linkUrl}
              onChange={(e) => setDraft((d) => ({ ...d, linkUrl: e.target.value }))} placeholder="https://pan.baidu.com/s/... 或 直链" />
          </div>
          <div className="admin-form-row">
            <div className="admin-field">
              <label>价格 (¥，0=免费)</label>
              <input className="admin-input" type="number" min="0" step="0.01" style={{ width: '100%' }}
                value={draft.price} onChange={(e) => setDraft((d) => ({ ...d, price: e.target.value }))} />
            </div>
            <div className="admin-field" style={{ display: 'flex', alignItems: 'flex-end' }}>
              <label className="admin-checkbox-row">
                <input type="checkbox" checked={!!draft.isMemberFree}
                  onChange={(e) => setDraft((d) => ({ ...d, isMemberFree: e.target.checked }))} />
                会员免费下载
              </label>
            </div>
          </div>
          <div className="row-actions">
            <button className="admin-btn sm" onClick={saveDraft}>保存资源</button>
            <button className="admin-btn ghost sm" onClick={() => setDraft(null)}>取消</button>
          </div>
        </div>
      )}
      {migTarget && (
        <BaiduMigrateDrawer
          attachment={migTarget}
          articleId={effectiveId}
          onClose={() => setMigTarget(null)}
          onFinished={() => { /* 保留 migTarget 状态以显示结果；用户点关闭后清除 */ }}
        />
      )}
    </div>
  )
}

// ============ 百度网盘「解析并迁移」抽屉（原样迁出）============
function BaiduMigrateDrawer({ attachment, articleId, onClose, onFinished }) {
  const item = attachment // alias inside
  const [cookie, setCookie] = useState('')
  const [savedCookie, setSavedCookie] = useState('')
  const [busy, setBusy] = useState('') // '' | 'resolve' | 'migrate'
  const [resolved, setResolved] = useState(null) // {files, isSelfShare, shareId, uk, title}
  const [migRec, setMigRec] = useState(null) // baidu_migrations 记录
  const [msg, setMsg] = useState('')

  const resolveNow = async () => {
    setMsg('')
    if (!item?.linkUrl) { setMsg('该资源缺少分享链接'); return }
    setBusy('resolve')
    try {
      const res = await adminApi.baidu.resolve({
        shareUrl: item.linkUrl, pwd: item.linkCode || '',
        articleId, attachmentId: item.id, cookie: cookie || undefined,
      })
      if (res.code !== 0) throw new Error(res.message || '解析失败')
      setResolved(res.data || {})
    } catch (e) { setMsg(e?.data?.message || e?.message || '解析失败') }
    finally { setBusy('') }
  }

  const migrateNow = async () => {
    setMsg('')
    setMigRec(null)
    if (!resolved || resolved.fileCount === 0) { setMsg('请先解析，确认已列出真实文件后再迁移'); return }
    setBusy('migrate')
    try {
      const res = await adminApi.baidu.migrate({
        shareUrl: item.linkUrl, pwd: item.linkCode || '',
        articleId, attachmentId: item.id, cookie: cookie || undefined,
      })
      if (res.code !== 0) throw new Error(res.message || '迁移失败')
      setMigRec({ status: res.data?.status, dlinkOk: res.data?.dlinkOk, panPath: res.data?.panPath,
        fileCount: res.data?.fileCount, fileNames: res.data?.fileNames || [], fileFsIds: res.data?.fileFsIds || [] })
      if (res.data?.dlinkOk) setSavedCookie(cookie)
      if (onFinished) setTimeout(onFinished, 300)
    } catch (e) { setMsg(e?.data?.message || e?.message || '迁移失败') }
    finally { setBusy('') }
  }

  const loadRec = async () => {
    try {
      const res = await adminApi.baidu.migrations(articleId)
      if (res.code === 0 && res.data?.list?.length) setMigRec(res.data.list[0])
    } catch (e) { /* ignore 读状态失败 */ }
  }
  useEffect(() => { if (articleId) loadRec() }, [articleId])

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,.45)', zIndex: 999,
      display: 'flex', alignItems: 'center', justifyContent: 'center' }} onClick={onClose}>
      <div className="item-form" style={{ width: 620, maxHeight: '82vh', overflow: 'auto', background: '#fff',
        borderRadius: 12, padding: 20 }} onClick={(e) => e.stopPropagation()}>
        <div className="module-editor-title">
          <i className="fa-solid fa-cloud-arrow-up" /> 百度网盘 · 解析并迁移
          <span className="admin-muted" style={{ marginLeft: 8, fontSize: 12 }}>资源：{item?.title}</span>
        </div>

        <div className="admin-hint" style={{ margin: '8px 0', lineHeight: 1.6 }}>
          <i className="fa-solid fa-circle-info" /> 迁移会把「分享内的文件」转到系统账号 <b>/gitmoe_files/{articleId}</b>，
          之后下载走本站代理直链，不再依赖原分享者。<br />
          系统配置的百度 cookie 会让百度返回「系统账号自己的根目录」，解析不出分享真实内容——
          <b>请粘贴一个能正常打开该分享的普通百度账号 cookie</b>（登录 pan.baidu.com → 复制 BDUSS/STOKEN 等完整 cookie 串）。
        </div>

        <div className="admin-field">
          <label>普通百度账号 Cookie（解析用，不会持久化，仅本次请求使用）</label>
          <textarea className="admin-input" style={{ width: '100%', minHeight: 64, resize: 'vertical' }}
            value={cookie} onChange={(e) => setCookie(e.target.value)}
            placeholder="BDUSS=xxx; STOKEN=yyy; ...（可选；留空则用系统账号，可能解析到根目录）" />
        </div>

        {msg && <div className="admin-error">{msg}</div>}

        <div className="row-actions" style={{ marginTop: 8 }}>
          <button className="admin-btn sm" disabled={busy !== ''} onClick={resolveNow}>
            {busy === 'resolve' ? '解析中…' : <><i className="fa-solid fa-magnifying-glass" /> 解析分享</>}
          </button>
          <button className="admin-btn sm" style={{ marginLeft: 6, background: '#2f7cf6' }}
            disabled={busy !== '' || !resolved || resolved.fileCount === 0} onClick={migrateNow}>
            {busy === 'migrate' ? '迁移中…' : <><i className="fa-solid fa-cloud-arrow-up" /> 迁移到系统账号</>}
          </button>
          <button className="admin-btn ghost sm" style={{ marginLeft: 6 }} onClick={onClose}>关闭</button>
        </div>

        {resolved && (
          <div style={{ marginTop: 12 }}>
            <div className="admin-muted">
              分享：{(resolved.title || '')} · {resolved.fileCount} 项
              {resolved.isSelf ? ' · 系统账号自分享' : ''}
            </div>
            <ul style={{ margin: '6px 0 0', paddingLeft: 18, fontSize: 13 }}>
              {(resolved.files || []).slice(0, 30).map((f) => (
                <li key={f.fsId} className="admin-muted">
                  {f.isDir ? '📁' : '📄'} {f.filename}
                  {f.dlinkOk ? '　<span className="admin-badge">直链OK</span>' : ''}
                </li>
              ))}
              {(resolved.files || []).length > 30 && <li className="admin-muted">… 共 {resolved.files.length} 项</li>}
            </ul>
          </div>
        )}

        {migRec && (
          <div style={{ marginTop: 14, padding: 10, background: '#f5f7fa', borderRadius: 8 }}>
            <div className="module-editor-title" style={{ marginBottom: 6 }}>
              迁移结果
              <span className={`admin-badge ${migRec.dlinkOk ? '' : 'gray'}`} style={{ marginLeft: 8 }}>
                {migRec.dlinkOk ? '✅ 已成功（直链可用）' : (migRec.status || '进行中')}
              </span>
            </div>
            <div className="admin-muted">路径：{migRec.panPath} · 文件数：{migRec.fileCount}</div>
            {migRec.fileNames?.length > 0 && (
              <div className="admin-muted" style={{ marginTop: 4 }}>
                文件：{migRec.fileNames.join('、')}
              </div>
            )}
            {!migRec.dlinkOk && migRec.status === 'failed' && <div className="admin-error">迁移失败，请检查 cookie 是否有效、分享是否可访问。</div>}
          </div>
        )}
      </div>
    </div>
  )
}

// ============ 下载模块元框：文章级字段 + 资源列表 ============
const PLATFORM_OPTS = PLATFORM_ORDER.map((pv) => ({ v: pv, label: PLATFORM_LABELS[pv] || pv }))

export default function DownloadMetaBox({ form, onChange, errors, articleId, ensureArticleId }) {
  return (
    <Metabox title="资源信息" icon="fa-download">
      <Field label="语言" hint="资源界面/字幕语言">
        <Select value={form.language} onChange={(v) => onChange({ language: v })} options={LANGUAGES} />
      </Field>
      <Field label="游戏设备" required error={errors?.gamePlatform}>
        <Select value={form.gamePlatform} onChange={(v) => onChange({ gamePlatform: v })} options={PLATFORM_OPTS} placeholder="请选择（下载模块必填）" />
      </Field>
      <Field label="游戏类型（可多选）">
        <div className="genre-chips">
          {['SLG', 'RPG', 'ADV', 'ACT', 'HTML', '其他'].map((g) => {
            const cur = (form.gameGenre || '').split(',').map((x) => x.trim()).filter(Boolean)
            const on = cur.includes(g)
            return (
              <button type="button" key={g} className={'genre-chip' + (on ? ' active' : '')}
                onClick={() => {
                  const next = on ? cur.filter((x) => x !== g) : [...cur, g]
                  onChange({ gameGenre: next.join(',') })
                }}>{g}</button>
            )
          })}
        </div>
      </Field>
      <Field label="资源版本" hint="如 v1.2.3">
        <TextInput value={form.resourceVersion} onChange={(v) => onChange({ resourceVersion: v })} placeholder="如 1.2.3" />
      </Field>
      <Field label="出品方/开发商">
        <TextInput value={form.publisher} onChange={(v) => onChange({ publisher: v })} />
      </Field>
      <Field label="资源总大小（展示）" hint="如 12.3GB">
        <TextInput value={form.resourceSizeDisplay} onChange={(v) => onChange({ resourceSizeDisplay: v })} placeholder="如 12.3GB" />
      </Field>
      <Field label="运行环境/最低配置">
        <TextArea value={form.requirement} onChange={(v) => onChange({ requirement: v })} rows={3} placeholder="系统要求、依赖等" />
      </Field>
      <Field label="资源更新时间">
        <DateInput value={form.resourceUpdatedAt || ''} onChange={(v) => onChange({ resourceUpdatedAt: v })} />
      </Field>
      <Switch checked={!!form.isOfficial} onChange={(v) => onChange({ isOfficial: v })} label=" 官方正版资源" />

      <div className="module-section-divider" />
      <DownloadEditor articleId={articleId} ensureArticleId={ensureArticleId} />
    </Metabox>
  )
}
