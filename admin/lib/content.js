// ★ 修改时请同步 D:\gm\web-next\lib\content.js（两份 isHtmlContent 必须逐字节一致）
//
// 后台侧正文格式单一来源：isHtmlContent / markdownToHtml / pickRenderedHtml。
// 前台 web-next/lib/content.js 是同一份逻辑的副本（两个独立 npm 项目，无法 import 同一文件）。

import { marked } from 'marked'

export function isHtmlContent(text) {
  if (!text) return false
  const t = text.trim()
  return t.startsWith('<') && t.includes('>')
}

export function markdownToHtml(md) {
  if (!md) return ''
  return marked.parse(md)
}

// 正文渲染分流核心：优先按 content_format；'' 时走启发式兜底（与旧行为一致）。
export function pickRenderedHtml(content, contentFormat) {
  if (contentFormat === 'html') return content
  if (contentFormat === 'markdown') return markdownToHtml(content)
  return isHtmlContent(content) ? content : markdownToHtml(content)
}
