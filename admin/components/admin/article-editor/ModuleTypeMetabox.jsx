'use client'

import Metabox from './Metabox'
import { VISIBLE_CONTENT_TYPES } from './constants'

// 模块类型元框：下拉切换（带二次确认，由屏幕的 onSwitchType 处理）。
export default function ModuleTypeMetabox({ form, onSwitchType }) {
  return (
    <Metabox title="模块类型" icon="fa-layer-group">
      <div className="admin-field">
        <select className="admin-select" style={{ width: '100%' }}
          value={form.contentType}
          onChange={(e) => onSwitchType(e.target.value)}>
          {VISIBLE_CONTENT_TYPES.map((t) => (
            <option key={t.v} value={t.v}>{t.label}（{t.desc}）</option>
          ))}
        </select>
        <div className="admin-muted" style={{ fontSize: 12, marginTop: 4 }}>
          未选择时默认为「新闻模块」
        </div>
      </div>
    </Metabox>
  )
}
