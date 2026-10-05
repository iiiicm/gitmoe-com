'use client'

import Metabox from './Metabox'
import { Field, TextInput, Select, Switch } from './fields'
import { SOURCE_TYPES } from './constants'

// 新闻（article）模块专属属性：副标题/来源/头条/评论——补齐原「纯文章」缺失的 6 个字段。
export default function ArticleMetaBox({ form, onChange }) {
  return (
    <Metabox title="新闻属性" icon="fa-newspaper">
      <Field label="副标题/导语">
        <TextInput value={form.subtitle} onChange={(v) => onChange({ subtitle: v })} placeholder="可选，显示在标题下方" />
      </Field>
      <Field label="内容来源类型">
        <Select value={form.sourceType} onChange={(v) => onChange({ sourceType: v })} options={SOURCE_TYPES} />
      </Field>
      <Field label="来源/转载出处">
        <TextInput value={form.sourceName} onChange={(v) => onChange({ sourceName: v })} placeholder="如：某汉化组" />
      </Field>
      <Field label="原文链接">
        <TextInput value={form.sourceUrl} onChange={(v) => onChange({ sourceUrl: v })} placeholder="https://..." />
      </Field>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <Switch checked={!!form.isFeatured} onChange={(v) => onChange({ isFeatured: v })} label=" 设为头条/置顶" />
        <Switch checked={!!form.allowComment} onChange={(v) => onChange({ allowComment: v })} label=" 允许评论" />
      </div>
    </Metabox>
  )
}
