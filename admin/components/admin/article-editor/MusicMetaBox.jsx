'use client'

import { useEffect, useState, useCallback } from 'react'
import { adminApi } from '../../../lib/adminApi'
import Metabox from './Metabox'

// ============ 音乐模块编辑器：免费音轨（原样迁出；★ 当前未挂载，music 已下线）============
// TODO: music 暂时下线（MUSIC_MODULE_ENABLED=false）。恢复时把 constants.js 的开关改为 true，
//       并在 ArticleEditorScreen 的 TYPE_METABOX_MAP 中启用本组件即可，无需改内部逻辑。
function MusicEditor({ articleId, ensureArticleId }) {
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
      if (res.code === 0) setItems((res.data || []).filter((x) => x.linkType === 'music'))
    } catch (e) { setMsg(e?.data?.message || e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [effectiveId])
  useEffect(() => { load() }, [load])

  const newDraft = () => setDraft({ id: null, title: '', linkUrl: '' })
  const editDraft = (it) => setDraft({ id: it.id, title: it.title, linkUrl: it.linkUrl || '' })

  const saveDraft = async () => {
    setMsg('')
    if (!draft.title.trim()) { setMsg('请填写曲目名称'); return }
    if (!draft.linkUrl.trim()) { setMsg('请填写音频链接'); return }
    try {
      if (draft.id) {
        const res = await adminApi.attachments.update(draft.id, { title: draft.title, linkUrl: draft.linkUrl })
        if (res.code !== 0) throw new Error(res.message || '更新失败')
      } else {
        const fd = new FormData()
        fd.append('title', draft.title)
        fd.append('linkType', 'music')
        fd.append('linkUrl', draft.linkUrl)
        fd.append('price', 0)
        fd.append('isMemberFree', 'true')
        const res = await adminApi.attachments.create(effectiveId, fd)
        if (res.code !== 0) throw new Error(res.message || '添加失败')
      }
      setDraft(null); load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '保存失败') }
  }
  const del = async (id) => {
    if (!confirm('确认删除该音轨？')) return
    const res = await adminApi.attachments.remove(id)
    if (res.code === 0) load(); else setMsg(res.message || '删除失败')
  }

  const uploadLocalMusic = async (file, id) => {
    if (!file) return
    setMsg('')
    try {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('type', 'music')
      const up = await adminApi.articles.uploadGeneric(fd)
      if (up.code !== 0) throw new Error(up.message || '上传失败')
      const url = up.data?.url
      if (!url) throw new Error('上传未返回地址')
      const createFd = new FormData()
      createFd.append('title', file.name)
      createFd.append('linkType', 'music')
      createFd.append('linkUrl', url)
      createFd.append('price', 0)
      createFd.append('isMemberFree', 'true')
      const res = await adminApi.attachments.create(id, createFd)
      if (res.code !== 0) throw new Error(res.message || '添加失败')
      load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '上传失败') }
  }

  return (
    <div className="module-editor">
      {!effectiveId && (
        <div className="admin-hint admin-hint-locked">
          <i className="fa-solid fa-circle-info" /> 首次添加音轨将自动创建文章草稿，无需先点「保存」。
        </div>
      )}
      <div className="module-editor-title"><i className="fa-solid fa-music" /> 音乐音轨（纯免费展示）</div>
      {msg && <div className="admin-error">{msg}</div>}
      {loading && <div className="admin-muted">加载中…</div>}
      <div className="item-list">
        {items.length === 0 && !loading && <div className="admin-muted">暂无音轨，请在下方添加。</div>}
        {items.map((it) => (
          <div className="item-row" key={it.id}>
            <div className="item-main">
              <div className="item-name">{it.title}</div>
              <div className="admin-muted item-sub">{it.linkUrl}</div>
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
          <button className="admin-btn ghost sm" onClick={async () => { const id = await ensureId(); if (id) newDraft() }}><i className="fa-solid fa-plus" /> 添加音轨</button>
          <label className="admin-btn ghost sm" style={{ cursor: 'pointer' }}>
            <i className="fa-solid fa-cloud-arrow-up" /> 上传本地音频
            <input type="file" accept="audio/*" style={{ display: 'none' }} onChange={async (e) => { const id = await ensureId(); if (id) uploadLocalMusic(e.target.files[0], id); e.target.value = '' }} />
          </label>
        </div>
      )}
      {draft && (
        <div className="item-form">
          <div className="admin-form-row">
            <div className="admin-field" style={{ flex: 2 }}>
              <label>曲目名称</label>
              <input className="admin-input" style={{ width: '100%' }} value={draft.title}
                onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))} placeholder="如：主题曲 / OP1" />
            </div>
          </div>
          <div className="admin-field">
            <label>音频链接（mp3 等）</label>
            <input className="admin-input" style={{ width: '100%' }} value={draft.linkUrl}
              onChange={(e) => setDraft((d) => ({ ...d, linkUrl: e.target.value }))} placeholder="https://.../track.mp3" />
          </div>
          <div className="row-actions">
            <button className="admin-btn sm" onClick={saveDraft}>保存音轨</button>
            <button className="admin-btn ghost sm" onClick={() => setDraft(null)}>取消</button>
          </div>
        </div>
      )}
    </div>
  )
}

// 音乐模块元框（保留代码，默认不渲染，由 MUSIC_MODULE_ENABLED 控制）。
export default function MusicMetaBox({ articleId, ensureArticleId }) {
  return (
    <Metabox title="音乐信息" icon="fa-music">
      <MusicEditor articleId={articleId} ensureArticleId={ensureArticleId} />
    </Metabox>
  )
}
