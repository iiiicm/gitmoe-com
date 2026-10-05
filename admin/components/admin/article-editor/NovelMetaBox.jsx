'use client'

import Metabox from './Metabox'
import { Field, TextInput, NumberInput, Select } from './fields'
import { SERIAL_STATUS, REGIONS } from './constants'
import ChapterEditor from './ChapterEditor'

// 小说模块元框：文章级字段 + 章节（正文 + TXT 导入）。
export default function NovelMetaBox({ form, onChange, articleId, ensureArticleId }) {
  return (
    <Metabox title="小说信息" icon="fa-book">
      <div className="admin-form-row">
        <Field label="连载状态">
          <Select value={form.serialStatus} onChange={(v) => onChange({ serialStatus: v })} options={SERIAL_STATUS} />
        </Field>
        <Field label="地区">
          <Select value={form.region} onChange={(v) => onChange({ region: v })} options={REGIONS} />
        </Field>
      </div>
      <Field label="首发平台">
        <TextInput value={form.firstPlatform} onChange={(v) => onChange({ firstPlatform: v })} placeholder="如 起点 / 晋江" />
      </Field>
      <Field label="字数">
        <NumberInput value={form.wordCount} onChange={(v) => onChange({ wordCount: v })} placeholder="如 1200000" />
      </Field>

      <div className="module-section-divider" />
      <ChapterEditor articleId={articleId} contentType="novel" ensureArticleId={ensureArticleId} />
    </Metabox>
  )
}
