'use client'

import { useEffect, useState, useCallback } from 'react'
import { adminApi } from '../../../lib/adminApi'
import Metabox from './Metabox'
import { Field, TextInput, TextArea, NumberInput, DateInput } from './fields'
import { VIDEO_TYPES } from './constants'

// ============ 视频模块编辑器：分集视频源 + 粘贴链接自动抓封面/标题（原样迁出）============
function VideoEditor({ articleId, ensureArticleId }) {
  const [effectiveId, setEffectiveId] = useState(articleId)
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [draft, setDraft] = useState(null)
  const [parsing, setParsing] = useState(false)
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
      const res = await adminApi.videos.list(effectiveId)
      if (res.code === 0) setItems(res.data || [])
    } catch (e) { setMsg(e?.data?.message || e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [effectiveId])
  useEffect(() => { load() }, [load])

  const newDraft = () => setDraft({ id: null, title: '', sourceType: 'mp4', sourceUrl: '', thumbnail: '', duration: 0 })
  const editDraft = (it) => setDraft({ id: it.id, title: it.title || '', sourceType: it.sourceType || 'mp4', sourceUrl: it.sourceUrl || '', thumbnail: it.thumbnail || '', duration: it.duration || 0 })

  const autoFetch = async () => {
    if (!draft.sourceUrl.trim()) { setMsg('请先粘贴视频链接'); return }
    setParsing(true); setMsg('')
    try {
      const res = await adminApi.videos.parseInfo(draft.sourceUrl.trim())
      if (res.code === 0 && res.data) {
        setDraft((d) => ({
          ...d,
          title: res.data.title || d.title,
          thumbnail: res.data.thumbnail || d.thumbnail,
          sourceType: res.data.sourceType || d.sourceType,
        }))
      } else { setMsg('解析失败：' + (res.message || '无法获取封面/标题')) }
    } catch (e) { setMsg(e?.data?.message || e?.message || '解析失败') }
    finally { setParsing(false) }
  }

  const saveDraft = async () => {
    setMsg('')
    if (!draft.sourceUrl.trim()) { setMsg('请填写视频链接'); return }
    try {
      if (draft.id) {
        const res = await adminApi.videos.update(draft.id, {
          title: draft.title, sourceType: draft.sourceType, sourceUrl: draft.sourceUrl,
          thumbnail: draft.thumbnail, duration: Number(draft.duration) || 0,
        })
        if (res.code !== 0) throw new Error(res.message || '更新失败')
      } else {
        const res = await adminApi.videos.add(effectiveId, {
          title: draft.title, sourceType: draft.sourceType, sourceUrl: draft.sourceUrl,
          thumbnail: draft.thumbnail, duration: Number(draft.duration) || 0,
        })
        if (res.code !== 0) throw new Error(res.message || '添加失败')
      }
      setDraft(null); load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '保存失败') }
  }
  const del = async (id) => {
    if (!confirm('确认删除该视频源？')) return
    const res = await adminApi.videos.remove(id)
    if (res.code === 0) load(); else setMsg(res.message || '删除失败')
  }

  const uploadLocalVideo = async (file, id) => {
    if (!file) return
    setMsg('')
    try {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('type', 'video')
      const up = await adminApi.articles.uploadGeneric(fd)
      if (up.code !== 0) throw new Error(up.message || '上传失败')
      const url = up.data?.url
      if (!url) throw new Error('上传未返回地址')
      const res = await adminApi.videos.add(id, {
        title: file.name,
        sourceType: 'mp4',
        sourceUrl: url,
        thumbnail: '',
        duration: 0,
      })
      if (res.code !== 0) throw new Error(res.message || '添加失败')
      load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '上传失败') }
  }

  return (
    <div className="module-editor">
      {!effectiveId && (
        <div className="admin-hint admin-hint-locked">
          <i className="fa-solid fa-circle-info" /> 首次添加视频源将自动创建文章草稿，无需先点「保存」。
        </div>
      )}
      <div className="module-editor-title"><i className="fa-solid fa-film" /> 视频源（按集添加，支持 mp4 / m3u8 / X.com / YouTube / Bilibili）</div>
      {msg && <div className="admin-error">{msg}</div>}
      {loading && <div className="admin-muted">加载中…</div>}
      <div className="item-list">
        {items.length === 0 && !loading && <div className="admin-muted">暂无视频源，请在下方添加。</div>}
        {items.map((it) => (
          <div className="item-row" key={it.id}>
            <div className="item-main">
              <div className="item-name">{it.title || '（未命名）'} <span className="admin-badge gray">{it.sourceType}</span> {it.thumbnail ? <span className="admin-badge green">有封面</span> : null}</div>
              <div className="admin-muted item-sub">{it.sourceUrl}</div>
            </div>
            <div className="row-actions">
              <button className="link-btn" onClick={() => editDraft(it)}>编辑</button>
              <button className="link-btn danger" onClick={() => del(it.id)}>删除</button>
            </div>
          </div>
        ))}
      </div>

      {!draft && (
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <button className="admin-btn ghost sm" onClick={async () => { const id = await ensureId(); if (id) newDraft() }}><i className="fa-solid fa-plus" /> 添加视频源</button>
          <label className="admin-btn ghost sm" style={{ cursor: 'pointer' }}>
            <i className="fa-solid fa-cloud-arrow-up" /> 上传本地视频
            <input type="file" accept="video/*" style={{ display: 'none' }} onChange={async (e) => { const id = await ensureId(); if (id) uploadLocalVideo(e.target.files[0], id); e.target.value = '' }} />
          </label>
        </div>
      )}
      {draft && (
        <div className="item-form">
          <div className="admin-form-row">
            <div className="admin-field" style={{ flex: 2 }}>
              <label>集数 / 标题</label>
              <input className="admin-input" style={{ width: '100%' }} value={draft.title}
                onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))} placeholder="如：第1集 / PV" />
            </div>
            <div className="admin-field">
              <label>视频类型</label>
              <select className="admin-select" style={{ width: '100%' }} value={draft.sourceType}
                onChange={(e) => setDraft((d) => ({ ...d, sourceType: e.target.value }))}>
                {VIDEO_TYPES.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
              </select>
            </div>
          </div>
          <div className="admin-field">
            <label>视频链接</label>
            <div style={{ display: 'flex', gap: 8 }}>
              <input className="admin-input" style={{ flex: 1 }} value={draft.sourceUrl}
                onChange={(e) => setDraft((d) => ({ ...d, sourceUrl: e.target.value }))}
                placeholder="粘贴 mp4 / m3u8 / x.com / youtube 链接" />
              <button className="admin-btn ghost sm" onClick={autoFetch} disabled={parsing}>
                {parsing ? '解析中…' : '自动获取封面/标题'}
              </button>
            </div>
            <div className="admin-muted" style={{ fontSize: 12, marginTop: 4 }}>
              支持 X.com / YouTube / Bilibili：粘贴链接后点上方按钮，自动回填标题与封面。
            </div>
          </div>
          <div className="admin-form-row">
            <div className="admin-field" style={{ flex: 2 }}>
              <label>封面图 URL</label>
              <input className="admin-input" style={{ width: '100%' }} value={draft.thumbnail}
                onChange={(e) => setDraft((d) => ({ ...d, thumbnail: e.target.value }))} placeholder="自动获取或手动填写" />
            </div>
            <div className="admin-field">
              <label>时长(秒)</label>
              <input className="admin-input" type="number" min="0" style={{ width: '100%' }}
                value={draft.duration} onChange={(e) => setDraft((d) => ({ ...d, duration: e.target.value }))} />
            </div>
          </div>
          <div className="row-actions">
            <button className="admin-btn sm" onClick={saveDraft}>保存视频源</button>
            <button className="admin-btn ghost sm" onClick={() => setDraft(null)}>取消</button>
          </div>
        </div>
      )}
    </div>
  )
}

// ============ 视频模块元框：文章级字段 + 分集播放源 ============
export default function VideoMetaBox({ form, onChange, articleId, ensureArticleId }) {
  return (
    <Metabox title="视频信息" icon="fa-film">
      <div className="admin-form-row">
        <Field label="总时长（分钟）">
          <NumberInput value={form.durationMinutes} onChange={(v) => onChange({ durationMinutes: v })} />
        </Field>
        <Field label="集数">
          <NumberInput value={form.episodeTotal} onChange={(v) => onChange({ episodeTotal: v })} />
        </Field>
        <Field label="上映年份">
          <NumberInput value={form.releaseYear} onChange={(v) => onChange({ releaseYear: v })} />
        </Field>
      </div>
      <Field label="上映日期">
        <DateInput value={form.releaseDate || ''} onChange={(v) => onChange({ releaseDate: v })} />
      </Field>
      <Field label="主演/声优">
        <TextInput value={form.cast} onChange={(v) => onChange({ cast: v })} />
      </Field>
      <Field label="导演">
        <TextInput value={form.director} onChange={(v) => onChange({ director: v })} />
      </Field>
      <Field label="清晰度选项（多选，逗号分隔）" hint="本片提供哪些清晰度，如 1080P,720P">
        <TextInput value={form.qualityOptions} onChange={(v) => onChange({ qualityOptions: v })} placeholder="如 1080P,720P" />
      </Field>

      <div className="module-section-divider" />
      <VideoEditor articleId={articleId} ensureArticleId={ensureArticleId} />
    </Metabox>
  )
}
