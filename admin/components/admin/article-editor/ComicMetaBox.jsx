'use client'

import Metabox from './Metabox'
import { Field, TextInput, NumberInput, Select } from './fields'
import { SERIAL_STATUS, REGIONS, TRANSLATION_STATUS } from './constants'
import ChapterEditor from './ChapterEditor'

// 漫画模块元框：文章级字段 + 章节（话 + 图片列表）。
export default function ComicMetaBox({ form, onChange, articleId, ensureArticleId }) {
  return (
    <Metabox title="漫画信息" icon="fa-book-open">
      <div className="admin-form-row">
        <Field label="连载状态">
          <Select value={form.serialStatus} onChange={(v) => onChange({ serialStatus: v })} options={SERIAL_STATUS} />
        </Field>
        <Field label="地区">
          <Select value={form.region} onChange={(v) => onChange({ region: v })} options={REGIONS} />
        </Field>
      </div>
      <Field label="作画">
        <TextInput value={form.illustrator} onChange={(v) => onChange({ illustrator: v })} />
      </Field>
      <div className="admin-form-row">
        <Field label="话数">
          <NumberInput value={form.chapterTotal} onChange={(v) => onChange({ chapterTotal: v })} />
        </Field>
        <Field label="首发平台">
          <TextInput value={form.firstPlatform} onChange={(v) => onChange({ firstPlatform: v })} placeholder="如 哔哩哔哩漫画" />
        </Field>
      </div>
      <Field label="汉化状态">
        <Select value={form.translationStatus} onChange={(v) => onChange({ translationStatus: v })} options={TRANSLATION_STATUS} />
      </Field>

      <div className="module-section-divider" />
      <ChapterEditor articleId={articleId} contentType="comic" ensureArticleId={ensureArticleId} />
    </Metabox>
  )
}
