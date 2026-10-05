'use client'

import Metabox from './Metabox'
import { Field } from './fields'

// 分类 + 标签元框。
export default function TaxonomyMetabox({ form, onChange, cats }) {
  return (
    <Metabox title="分类" icon="fa-folder">
      <Field label="分类">
        <select className="admin-select" style={{ width: '100%' }}
          value={form.categoryId}
          onChange={(e) => onChange({ categoryId: e.target.value })}>
          <option value="">未分类</option>
          {(cats || []).map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
      </Field>
      <Field label="标签（逗号分隔）">
        <input className="admin-input" style={{ width: '100%' }}
          value={form.tags}
          onChange={(e) => onChange({ tags: e.target.value })} placeholder="如：汉化,合集" />
      </Field>
    </Metabox>
  )
}
