'use client'

import { useEffect, useState, useRef, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import dynamic from 'next/dynamic'

// TinyMCE 依赖 window/document，必须 ssr:false。
const RichTextEditor = dynamic(() => import('../RichTextEditor'), {
  ssr: false,
  loading: () => <div className="admin-editor-rtf-loading">编辑器加载中…</div>,
})

import { adminApi } from '../../../lib/adminApi'
import { isHtmlContent, markdownToHtml } from '../../../lib/content'
import { PAID_MODULES, MUSIC_MODULE_ENABLED } from './constants'
import { emptyForm, formFromArticle } from './defaults'
import { buildSavePayload } from './payload'
import { validateForm } from './validate'

import PublishMetabox from './PublishMetabox'
import ModuleTypeMetabox from './ModuleTypeMetabox'
import TaxonomyMetabox from './TaxonomyMetabox'
import CoverMetabox from './CoverMetabox'
import PaidMetabox from './PaidMetabox'
import ArticleMetaBox from './ArticleMetaBox'
import DownloadMetaBox from './DownloadMetaBox'
import VideoMetaBox from './VideoMetaBox'
import ComicMetaBox from './ComicMetaBox'
import NovelMetaBox from './NovelMetaBox'
import MusicMetaBox from './MusicMetaBox'

// 类型 → 元框组件查表（music 分支保留但默认不可达）。
const TYPE_METABOX_MAP = {
  article: ArticleMetaBox,
  download: DownloadMetaBox,
  video: VideoMetaBox,
  comic: ComicMetaBox,
  novel: NovelMetaBox,
  music: MusicMetaBox,
}

/**
 * 文章编辑器共享容器（新建 / 编辑共用）。
 * @param {'create'|'edit'} mode
 * @param {number|null} articleId
 */
export default function ArticleEditorScreen({ mode = 'create', articleId = null }) {
  const router = useRouter()
  const isEdit = mode === 'edit' && articleId != null

  const [form, setForm] = useState(emptyForm)
  const [snapshot, setSnapshot] = useState('')
  const snapshotRef = useRef('')
  const [dirty, setDirty] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [errors, setErrors] = useState({})
  const [topError, setTopError] = useState('')
  const [toast, setToast] = useState('')
  const [markdownNotice, setMarkdownNotice] = useState(false)
  const [cats, setCats] = useState([])
  const [levels, setLevels] = useState([])

  const setSnap = (s) => { snapshotRef.current = s; setSnapshot(s) }
  const patch = useCallback((p) => {
    setForm((prev) => {
      const nf = { ...prev, ...p }
      setDirty(JSON.stringify(nf) !== snapshotRef.current)
      return nf
    })
  }, [])

  // ---- 加载详情（编辑态）----
  useEffect(() => {
    let cancelled = false
    async function load() {
      if (mode !== 'edit' || articleId == null) {
        const f = emptyForm()
        setForm(f); setSnap(JSON.stringify(f)); setDirty(false)
        return
      }
      setLoading(true)
      try {
        const res = await adminApi.articles.get(articleId)
        if (cancelled) return
        if (res.code === 0) {
          const a = res.data.article || {}
          // 存量 Markdown 正文：编辑器内自动转 HTML 展示；保存后 content_format='html'
          let displayContent = a.content || ''
          let mdNotice = false
          if (a.contentFormat !== 'html' && !isHtmlContent(a.content)) {
            displayContent = markdownToHtml(a.content || '')
            mdNotice = !!(a.content && a.content.trim())
          }
          const f = formFromArticle(a)
          f.content = displayContent
          setForm(f)
          setSnap(JSON.stringify(f))
          setDirty(false)
          setMarkdownNotice(mdNotice)
        } else {
          setTopError(res.message || '加载失败')
        }
      } catch (e) {
        if (!cancelled) setTopError(e?.data?.message || e?.message || '加载失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [mode, articleId])

  // ---- 分类 + 会员等级（付费元框用）----
  useEffect(() => {
    adminApi.categories.list().then((c) => { if (c.code === 0) setCats(c.data || []) }).catch(() => {})
    adminApi.membership.list()
      .then((m) => { if (m.code === 0) setLevels((m.data || []).filter((l) => (l.key || '').toLowerCase() !== 'free')) })
      .catch(() => {})
  }, [])

  // ---- beforeunload 拦截 ----
  useEffect(() => {
    const handler = (e) => {
      if (dirty) { e.preventDefault(); e.returnValue = '' }
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  // ---- 首次保存自动建草稿拿 id（供子编辑器使用）----
  const ensureArticleId = useCallback(async () => {
    if (form.id) return form.id
    const payload = buildSavePayload(form)
    payload.status = 'draft'
    const res = await adminApi.articles.create(payload)
    if (res.code === 0 && res.data?.id) {
      const id = res.data.id
      setForm((f) => ({ ...f, id, status: 'draft' }))
      return id
    }
    throw new Error(res.message || '创建文章失败')
  }, [form])

  // ---- 保存 ----
  const save = useCallback(async (statusOverride) => {
    const { ok, errors: errs } = validateForm(form)
    setErrors(errs)
    if (!ok) {
      setTopError('请修正表单中标红的错误后再保存')
      return
    }
    setSaving(true)
    setTopError('')
    try {
      const payload = buildSavePayload(form)
      if (statusOverride) payload.status = statusOverride
      let res
      if (form.id) {
        res = await adminApi.articles.update(form.id, payload)
      } else {
        res = await adminApi.articles.create(payload)
      }
      if (res.code !== 0) {
        setTopError(res.message || '保存失败')
        return
      }
      if (!form.id && res.data?.id) {
        const id = res.data.id
        setForm((f) => ({ ...f, id, status: 'draft' }))
        setToast('已创建，现在可以添加模块资源了')
        setDirty(false)
        router.replace(`/articles/${id}/edit`)
        return
      }
      // 编辑态：重新加载详情，重置 dirty 与 snapshot
      const detail = await adminApi.articles.get(form.id)
      if (detail.code === 0) {
        const a = detail.data.article || {}
        let displayContent = a.content || ''
        if (a.contentFormat !== 'html' && !isHtmlContent(a.content)) {
          displayContent = markdownToHtml(a.content || '')
        }
        const f = formFromArticle(a)
        f.content = displayContent
        setForm(f)
        setSnap(JSON.stringify(f))
        setDirty(false)
      }
      setToast(statusOverride === 'draft' ? '草稿已保存' : '已保存')
    } catch (e) {
      setTopError(e?.data?.message || e?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }, [form, router])

  useEffect(() => {
    if (!toast) return
    const t = setTimeout(() => setToast(''), 2600)
    return () => clearTimeout(t)
  }, [toast])

  // ---- 切换模块类型（不清空已填字段，仅二次确认）----
  const switchType = (next) => {
    if (next === form.contentType) return
    if (dirty && !confirm('切换模块类型后，当前已填但未保存的字段仍会保留（只是不再显示），切换后请记得保存。确定切换？')) {
      return
    }
    patch({ contentType: next })
  }

  const isMusic = form.contentType === 'music'
  const TypeBox = TYPE_METABOX_MAP[form.contentType]
  const showPaid = PAID_MODULES.includes(form.contentType)

  return (
    <div className="admin-editor">
      <div className="admin-editor-topbar">
        <div className="admin-editor-topbar-left">
          <Link href="/articles" className="admin-editor-back">
            <i className="fa-solid fa-arrow-left" /> 所有文章
          </Link>
          <span className="admin-editor-crumb">
            <Link href="/articles">所有文章</Link>
            <span className="admin-editor-crumb-sep">/</span>
            <span>{isEdit ? '编辑文章' : '新建文章'}</span>
          </span>
        </div>
        <div className="admin-editor-topbar-right">
          <button className="admin-btn secondary sm" disabled title="草稿预览需后端临时 token（P1）">预览</button>
          <button className="admin-btn secondary sm" onClick={() => save('draft')} disabled={saving || isMusic}>
            保存草稿
          </button>
          <button className="admin-btn primary sm" onClick={() => save(form.status === 'draft' ? 'published' : form.status)} disabled={saving || isMusic}>
            {form.id ? '更新' : '发布'}
          </button>
        </div>
      </div>

      {topError && <div className="admin-editor-notice danger"><i className="fa-solid fa-circle-exclamation" /> {topError}</div>}
      {toast && <div className="admin-editor-notice success"><i className="fa-solid fa-circle-check" /> {toast}</div>}
      {markdownNotice && (
        <div className="admin-editor-notice warn" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span><i className="fa-solid fa-circle-info" /> 原文为 Markdown，已自动转换为富文本载入；保存后将以 HTML 格式存储。</span>
          <button className="admin-btn ghost sm" onClick={() => setMarkdownNotice(false)}>知道了</button>
        </div>
      )}
      {isMusic && (
        <div className="admin-editor-notice danger">
          <i className="fa-solid fa-ban" /> 音乐模块已暂时下线（只读）。本文章不可在后台编辑，仅前台可浏览。如需编辑请改模块类型或由管理员恢复音乐模块。
        </div>
      )}

      {loading && <div className="admin-editor-loading">加载中…</div>}

      <div className="admin-editor-body">
        <div className="admin-editor-main">
          <input
            className="admin-editor-title-input"
            placeholder="在此输入标题"
            value={form.title}
            disabled={isMusic}
            onChange={(e) => patch({ title: e.target.value })}
          />
          <div className="admin-editor-field-label">正文</div>
          <RichTextEditor key={`${mode}-${form.id}`} value={form.content} onChange={(html) => patch({ content: html })} />

          <div className="admin-editor-field-label">摘要</div>
          <textarea
            className="admin-textarea"
            rows={4}
            placeholder="摘要（选填，≤500 字）"
            value={form.summary}
            disabled={isMusic}
            onChange={(e) => patch({ summary: e.target.value })}
          />
        </div>

        <aside className="admin-editor-side">
          <PublishMetabox form={form} onChange={patch} onSave={save} saving={saving} />
          <ModuleTypeMetabox form={form} onSwitchType={switchType} />
          <TaxonomyMetabox form={form} onChange={patch} cats={cats} />
          <CoverMetabox form={form} onChange={patch} />
          {showPaid && <PaidMetabox form={form} onChange={patch} levels={levels} />}
          {TypeBox && !isMusic && (
            <TypeBox form={form} onChange={patch} errors={errors} articleId={form.id} ensureArticleId={ensureArticleId} />
          )}
        </aside>
      </div>
    </div>
  )
}
