'use client'

import { useEffect, useState, useCallback } from 'react'
import { adminApi } from '../../../lib/adminApi'

// ============ 漫画 / 小说 章节编辑器（原样迁出，isComic 由 contentType prop 驱动）============
export default function ChapterEditor({ articleId, contentType, ensureArticleId }) {
  const isComic = contentType === 'comic'
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
      const res = await adminApi.chapters.list(effectiveId)
      if (res.code === 0) setItems(res.data || [])
    } catch (e) { setMsg(e?.data?.message || e?.message || '加载失败') }
    finally { setLoading(false) }
  }, [effectiveId])
  useEffect(() => { load() }, [load])

  const newDraft = () => setDraft({ id: null, title: '', chapterOrder: items.length + 1, isFree: false, price: 0, content: '', images: '' })
  const editDraft = (it) => setDraft({ id: it.id, title: it.title || '', chapterOrder: it.chapterOrder || 1, isFree: !!it.isFree, price: it.price || 0, content: it.content || '', images: Array.isArray(it.images) ? it.images.join('\n') : (it.images || '') })

  const saveDraft = async () => {
    setMsg('')
    if (!draft.title.trim()) { setMsg('请填写章节名称'); return }
    const imagesArr = isComic
      ? draft.images.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
      : []
    try {
      if (draft.id) {
        const res = await adminApi.chapters.update(draft.id, {
          title: draft.title, chapterOrder: Number(draft.chapterOrder) || 1,
          isFree: !!draft.isFree, price: Number(draft.price) || 0,
          content: draft.content, images: JSON.stringify(imagesArr),
        })
        if (res.code !== 0) throw new Error(res.message || '更新失败')
      } else {
        const res = await adminApi.chapters.create({
          contentId: effectiveId, title: draft.title,
          chapterOrder: Number(draft.chapterOrder) || 1, isFree: !!draft.isFree,
          price: Number(draft.price) || 0, content: draft.content, images: JSON.stringify(imagesArr),
        })
        if (res.code !== 0) throw new Error(res.message || '添加失败')
      }
      setDraft(null); load()
    } catch (e) { setMsg(e?.data?.message || e?.message || '保存失败') }
  }
  const del = async (id) => {
    if (!confirm('确认删除该章节？')) return
    const res = await adminApi.chapters.remove(id)
    if (res.code === 0) load(); else setMsg(res.message || '删除失败')
  }

  const uploadComicImages = async (files) => {
    if (!files || files.length === 0) return
    setMsg('')
    const urls = []
    for (const file of Array.from(files)) {
      try {
        const fd = new FormData()
        fd.append('file', file)
        fd.append('type', 'comic')
        const up = await adminApi.articles.uploadGeneric(fd)
        if (up.code !== 0) throw new Error(up.message || '上传失败')
        if (up.data?.url) urls.push(up.data.url)
      } catch (e) { setMsg(e?.data?.message || e?.message || '部分图片上传失败'); break }
    }
    if (urls.length > 0) {
      setDraft((d) => ({ ...d, images: (d.images ? d.images + '\n' : '') + urls.join('\n') }))
    }
  }

  const uploadNovelText = async (file) => {
    if (!file) return
    try {
      const text = await file.text()
      setDraft((d) => ({ ...d, content: (d.content ? d.content + '\n\n' : '') + text }))
    } catch (e) { setMsg('读取文本文件失败') }
  }

  return (
    <div className="module-editor">
      {!effectiveId && (
        <div className="admin-hint admin-hint-locked">
          <i className="fa-solid fa-circle-info" /> 首次添加{isComic ? '漫画' : '小说'}章节将自动创建文章草稿，无需先点「保存」。
        </div>
      )}
      <div className="module-editor-title"><i className={isComic ? 'fa-solid fa-book-open' : 'fa-solid fa-book'} /> {isComic ? '漫画章节' : '小说章节'}</div>
      {msg && <div className="admin-error">{msg}</div>}
      {loading && <div className="admin-muted">加载中…</div>}
      <div className="item-list">
        {items.length === 0 && !loading && <div className="admin-muted">暂无章节，请在下方添加。</div>}
        {items.map((it) => (
          <div className="item-row" key={it.id}>
            <div className="item-main">
              <div className="item-name">第{it.chapterOrder}话/章 · {it.title} {it.isFree ? <span className="admin-badge green">免费</span> : (Number(it.price) > 0 ? <span className="admin-badge orange">¥{it.price}</span> : <span className="admin-badge gray">付费</span>)}</div>
            </div>
            <div className="row-actions">
              <button className="link-btn" onClick={() => editDraft(it)}>编辑</button>
              <button className="link-btn danger" onClick={() => del(it.id)}>删除</button>
            </div>
          </div>
        ))}
      </div>

      {!draft && <button className="admin-btn ghost sm" onClick={async () => { const id = await ensureId(); if (id) newDraft() }}><i className="fa-solid fa-plus" /> 添加章节</button>}
      {draft && (
        <div className="item-form">
          <div className="admin-form-row">
            <div className="admin-field" style={{ flex: 2 }}>
              <label>章节名称</label>
              <input className="admin-input" style={{ width: '100%' }} value={draft.title}
                onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))} placeholder="如：第1话 / 序章" />
            </div>
            <div className="admin-field">
              <label>章节序号</label>
              <input className="admin-input" type="number" min="1" style={{ width: '100%' }}
                value={draft.chapterOrder} onChange={(e) => setDraft((d) => ({ ...d, chapterOrder: e.target.value }))} />
            </div>
          </div>
          {isComic ? (
            <div className="admin-field">
              <label>漫画图片（每行一个图片 URL，或逗号分隔）</label>
              <textarea className="admin-textarea" rows={4} value={draft.images}
                onChange={(e) => setDraft((d) => ({ ...d, images: e.target.value }))} placeholder="https://.../page1.jpg&#10;https://.../page2.jpg" />
              <div style={{ marginTop: 8 }}>
                <label className="admin-btn ghost sm" style={{ cursor: 'pointer' }}>
                  <i className="fa-solid fa-cloud-arrow-up" /> 批量上传漫画图片
                  <input type="file" accept="image/*" multiple style={{ display: 'none' }} onChange={(e) => { uploadComicImages(e.target.files); e.target.value = '' }} />
                </label>
              </div>
            </div>
          ) : (
            <div className="admin-field">
              <label>小说正文</label>
              <textarea className="admin-textarea" rows={6} value={draft.content}
                onChange={(e) => setDraft((d) => ({ ...d, content: e.target.value }))} placeholder="章节正文内容（支持纯文本）" />
              <div style={{ marginTop: 8 }}>
                <label className="admin-btn ghost sm" style={{ cursor: 'pointer' }}>
                  <i className="fa-solid fa-cloud-arrow-up" /> 导入 TXT 文本
                  <input type="file" accept=".txt,text/plain" style={{ display: 'none' }} onChange={(e) => { uploadNovelText(e.target.files[0]); e.target.value = '' }} />
                </label>
              </div>
            </div>
          )}
          <div className="admin-form-row">
            <div className="admin-field">
              <label className="admin-checkbox-row">
                <input type="checkbox" checked={!!draft.isFree}
                  onChange={(e) => setDraft((d) => ({ ...d, isFree: e.target.checked }))} />
                免费章节（前几章建议设为免费试读）
              </label>
            </div>
            <div className="admin-field">
              <label>单章价格 (¥，0=按整体)</label>
              <input className="admin-input" type="number" min="0" step="0.01" style={{ width: '100%' }}
                value={draft.price} onChange={(e) => setDraft((d) => ({ ...d, price: e.target.value }))} />
            </div>
          </div>
          <div className="row-actions">
            <button className="admin-btn sm" onClick={saveDraft}>保存章节</button>
            <button className="admin-btn ghost sm" onClick={() => setDraft(null)}>取消</button>
          </div>
        </div>
      )}
    </div>
  )
}
