'use client'

import { useState } from 'react'

// 通用元框容器：标题栏点击折叠 + chevron 图标 + body 插槽。
// 发布框 collapsible={false}（对齐 WordPress 经典编辑器）。
export default function Metabox({ title, icon, collapsible = true, defaultOpen = true, footer, children }) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <section className={'admin-metabox' + (collapsible && !open ? ' admin-metabox-collapsed' : '')}>
      <div
        className="admin-metabox-head"
        onClick={collapsible ? () => setOpen((o) => !o) : undefined}
        style={collapsible ? { cursor: 'pointer' } : undefined}
      >
        <span className="admin-metabox-title">
          {icon && <i className={'fa-solid ' + icon} style={{ marginRight: 6 }} />}
          {title}
        </span>
        {collapsible && (
          <i className={'fa-solid fa-chevron-up' + (open ? '' : ' admin-rot')} />
        )}
      </div>
      {open && (
        <div className="admin-metabox-body">
          {children}
          {footer}
        </div>
      )}
    </section>
  )
}
