import { marked } from 'marked'
import DOMPurify from 'isomorphic-dompurify'
import { resolveContentImages } from './format'
import { pickRenderedHtml } from './content'

// 服务端/客户端通用的文章正文渲染：按 content_format 分流 Markdown/HTML → 净化 HTML。
// 服务端组件调用时由 isomorphic-dompurify（内置 jsdom）完成净化，确保正文进入 SSR 的 HTML 源码（SEO）。
// contentFormat 缺省（''）时走启发式兜底，与旧行为一致（兼容无该字段的存量数据）。
export function renderArticleBody(content, contentFormat) {
  if (!content) return ''
  const raw = pickRenderedHtml(content, contentFormat || '')
  const withImages = resolveContentImages(raw)
  return DOMPurify.sanitize(withImages)
}
