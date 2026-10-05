'use client'

// TinyMCE 6 封装（自托管 + 中文 + 粘贴/拖拽图片自动上传）。
//
// ★ 本组件依赖 window / document，App Router 下必须由调用方用
//   dynamic(() => import('.../RichTextEditor'), { ssr: false }) 引入，禁止直接静态 import。
//
// 上传契约：POST /api/admin/upload-generic（multipart，字段 file + type），
// 返回 { code:0, data:{ url,name,size,ext,type,local } }。

import { useRef } from 'react'
import { Editor } from '@tinymce/tinymce-react'
import { adminApi } from '../../lib/adminApi'

// 正文图片限制（与 PRD Q7 的客户端预校验口径一致；后端若更严会给出明确错误文案）
const MAX_IMAGE_BYTES = 10 * 1024 * 1024
const ALLOWED_IMAGE_EXT = ['jpg', 'jpeg', 'png', 'gif', 'webp']

function validateImage(blob, filename) {
  const ext = (filename || '').split('.').pop().toLowerCase()
  if (filename && !ALLOWED_IMAGE_EXT.includes(ext)) {
    return `不支持的图片格式 .${ext}，仅支持 ${ALLOWED_IMAGE_EXT.join(' / ')}`
  }
  if (blob && blob.size > MAX_IMAGE_BYTES) {
    return `图片超过 ${MAX_IMAGE_BYTES / 1024 / 1024}MB（当前 ${(blob.size / 1024 / 1024).toFixed(1)}MB）`
  }
  return ''
}

export default function RichTextEditor({ value = '', onChange, height = 420, disabled = false }) {
  // TinyMCE 的 onEditorChange 会在初始化时回调一次空内容，用 ref 挡掉这次「误清空」。
  const initedRef = useRef(false)

  /**
   * 图片上传钩子：覆盖「粘贴图片 / 拖拽图片 / 插入按钮」三条路径。
   * 成功 resolve(url)，失败 reject(Error) —— TinyMCE 会移除占位图并在编辑区顶部提示。
   * @param {Object} blobInfo TinyMCE 的 BlobInfo（.blob() / .filename()）
   * @returns {Promise<string>} 图片可访问 URL
   */
  const handleImageUpload = async (blobInfo) => {
    const blob = blobInfo.blob()
    const filename = blobInfo.filename()
    const why = validateImage(blob, filename)
    if (why) throw new Error(why)

    const fd = new FormData()
    fd.append('file', blob, filename || 'image.png')
    fd.append('type', 'attachments')

    const res = await adminApi.articles.uploadGeneric(fd)
    if (res && res.code === 0 && res.data && res.data.url) {
      return res.data.url
    }
    throw new Error((res && res.message) || '图片上传失败')
  }

  return (
    <div className="admin-editor-rtf">
      <Editor
        tinymceScriptSrc="/tinymce/tinymce.min.js"
        licenseKey="gpl"
        value={value}
        disabled={disabled}
        init={{
          // ---- 汉化与皮肤（自托管路径，见 scripts/copy-tinymce.mjs）----
          language: 'zh_CN',
          language_url: '/tinymce/langs/zh_CN.js',
          skin_url: '/tinymce/skins/ui/oxide',
          content_css: '/tinymce/skins/content/default/content.css',

          // ---- 外观（对齐 WordPress 经典编辑器）----
          promotion: false,
          branding: false,
          menubar: false,
          statusbar: true,
          min_height: height,
          toolbar_mode: 'wrap',
          toolbar_sticky: true,
          resize: 'vertical',
          // ★ 不要写 'paste'：TinyMCE 6 已把 paste 合入内核，写成插件会在控制台报“找不到插件”。
          plugins:
            'advlist autolink lists link image charmap preview anchor searchreplace visualblocks code fullscreen insertdatetime media table help wordcount',

          // ★ 两行工具栏必须用数组形式的 toolbar：tinymce@6.8.6 已不支持 v4 时代的 toolbar1/toolbar2
          //   （实测在 node_modules/tinymce/tinymce.js 里搜不到 "toolbar1"）。
          //   第一行常用；第二行点「工具栏切换 ▾」展开（WP 的 kitchen sink）。
          toolbar: [
            'formatselect | bold italic underline strikethrough | bullist numlist | blockquote | alignleft aligncenter alignright | link unlink | removeformat',
            'forecolor backcolor | hr charmap | indent outdent | superscript subscript | table | pastetext | code | fullscreen | help',
          ],

          // ---- 粘贴 / 上传 ----
          paste_data_images: true,
          automatic_uploads: true,
          images_reuse_filename: false,
          images_upload_handler: handleImageUpload,

          // 编辑区基础样式，让所见即所得更贴近前台正文
          content_style:
            'body { font-family: -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif; font-size: 15px; line-height: 1.8; } img { max-width: 100%; height: auto; }',
        }}
        onEditorChange={(html) => {
          if (!initedRef.current) {
            initedRef.current = true
            // 初始化回调里的内容与受控 value 一致时忽略，避免把已有正文冲掉
            if (html === (value || '')) return
          }
          if (typeof onChange === 'function') onChange(html || '')
        }}
      />
    </div>
  )
}
