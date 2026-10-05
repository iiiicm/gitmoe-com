'use client'

import Metabox from './Metabox'
import { Field, NumberInput, Switch } from './fields'
import { MEMBERS_ONLY_LABEL } from './constants'

// 付费设置元框：仅 PAID_MODULES 显示（由屏幕控制是否渲染）。
export default function PaidMetabox({ form, onChange, levels }) {
  const setVip = (key, val) =>
    onChange({ vipPrices: { ...(form.vipPrices || {}), [key]: val } })
  return (
    <Metabox title="付费设置" icon="fa-tags">
      <div className="admin-form-row">
        <Field label="注册用户价格 (¥)">
          <NumberInput value={form.price} onChange={(v) => onChange({ price: v })} />
        </Field>
        <Field label="原价 (¥)">
          <NumberInput value={form.originalPrice} onChange={(v) => onChange({ originalPrice: v })} />
        </Field>
      </div>
      <div className="admin-subtitle">各会员价格（默认 0，即该等级免费）</div>
      <div className="admin-form-row">
        {(!levels || levels.length === 0) ? (
          <div className="admin-muted" style={{ fontSize: 12 }}>暂无会员等级，仅按注册用户价格收费。</div>
        ) : levels.map((l) => (
          <Field key={l.key} label={l.name + ' 价 (¥)'}>
            <NumberInput value={form.vipPrices?.[l.key] ?? 0} onChange={(v) => setVip(l.key, v)} />
          </Field>
        ))}
      </div>
      <div style={{ marginTop: 8 }}>
        <Switch checked={!!form.membersOnly}
          onChange={(v) => onChange({ membersOnly: v })}
          label={' ' + (MEMBERS_ONLY_LABEL[form.contentType] || '仅会员可访问') + '（开启后仅有效会员可访问）'} />
      </div>
      {form.contentType === 'download' && (
        <Field label="下载模块标题（可选）" hint="留空则按价格自动显示「付费资源/免费资源」">
          <input className="admin-input" type="text" maxLength={100} style={{ width: '100%' }}
            value={form.downloadTitle || ''}
            onChange={(e) => onChange({ downloadTitle: e.target.value })} />
        </Field>
      )}
    </Metabox>
  )
}
