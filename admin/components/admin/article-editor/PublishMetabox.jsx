'use client'

import Metabox from './Metabox'
import { STATUS_OPTS } from './constants'

// 发布元框：状态切换 + 保存草稿 / 发布（更新）。
export default function PublishMetabox({ form, onChange, onSave, saving }) {
  return (
    <Metabox title="发布" icon="fa-paper-plane" collapsible={false}>
      <div className="admin-field">
        <label>状态</label>
        <select className="admin-select" style={{ width: '100%' }}
          value={form.status}
          onChange={(e) => onChange({ status: e.target.value })}>
          {STATUS_OPTS.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
        </select>
      </div>
      <div className="admin-metabox-actions">
        <button className="admin-btn ghost sm" type="button"
          onClick={() => onSave('draft')} disabled={saving}>保存草稿</button>
        <button className="admin-btn primary sm" type="button"
          onClick={() => onSave(form.status === 'draft' ? 'published' : form.status)} disabled={saving}>
          {form.id ? '更新' : '发布'}
        </button>
      </div>
    </Metabox>
  )
}
