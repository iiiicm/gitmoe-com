'use client'

import { useState, useRef } from 'react'
import { adminApi } from '../../../lib/adminApi'

// 表单原子组件库：所有元框必须复用，禁止各自手写 input，保证样式与交互统一。

export function Field({ label, hint, required, error, children }) {
  return (
    <div className="admin-field">
      {label && (
        <label>
          {label}
          {required && <span className="admin-req"> *</span>}
          {hint && <span className="admin-hint-inline">{hint}</span>}
        </label>
      )}
      {children}
      {error && <div className="admin-field-error">{error}</div>}
    </div>
  )
}

export function TextInput({ value, onChange, placeholder, type = 'text', ...rest }) {
  return (
    <input
      className="admin-input"
      type={type}
      style={{ width: '100%' }}
      value={value ?? ''}
      placeholder={placeholder}
      onChange={(e) => onChange && onChange(e.target.value)}
      {...rest}
    />
  )
}

export function TextArea({ value, onChange, rows = 3, placeholder }) {
  return (
    <textarea
      className="admin-textarea"
      rows={rows}
      value={value ?? ''}
      placeholder={placeholder}
      onChange={(e) => onChange && onChange(e.target.value)}
    />
  )
}

export function NumberInput({ value, onChange, min = 0, step = 1, placeholder }) {
  return (
    <input
      className="admin-input"
      type="number"
      min={min}
      step={step}
      style={{ width: '100%' }}
      value={value ?? ''}
      placeholder={placeholder}
      onChange={(e) => onChange && onChange(e.target.value === '' ? '' : Number(e.target.value))}
    />
  )
}

export function Select({ value, onChange, options, placeholder }) {
  return (
    <select
      className="admin-select"
      style={{ width: '100%' }}
      value={value ?? ''}
      onChange={(e) => onChange && onChange(e.target.value)}
    >
      {placeholder && <option value="">{placeholder}</option>}
      {options.map((o) => (
        <option key={o.v} value={o.v}>{o.label}</option>
      ))}
    </select>
  )
}

export function Switch({ checked, onChange, label }) {
  return (
    <label className="admin-checkbox-row">
      <input
        type="checkbox"
        checked={!!checked}
        onChange={(e) => onChange && onChange(e.target.checked)}
      />
      {label}
    </label>
  )
}

export function Chips({ value, onChange, options }) {
  const cur = (value || '').split(',').map((x) => x.trim()).filter(Boolean)
  const toggle = (g) => {
    const next = cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]
    onChange && onChange(next.join(','))
  }
  return (
    <div className="genre-chips">
      {options.map((g) => (
        <button
          type="button"
          key={g}
          className={'genre-chip' + (cur.includes(g) ? ' active' : '')}
          onClick={() => toggle(g)}
        >
          {g}
        </button>
      ))}
    </div>
  )
}

export function DateInput({ value, onChange }) {
  return (
    <input
      className="admin-input"
      type="date"
      style={{ width: '100%' }}
      value={value ?? ''}
      onChange={(e) => onChange && onChange(e.target.value)}
    />
  )
}

export function UploadField({ label, accept, type = 'attachments', value, onChange, placeholder = '或粘贴 URL', uploadText = '上传文件' }) {
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef(null)

  const handleFile = async (file) => {
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('type', type)
      const res = await adminApi.articles.uploadGeneric(fd)
      if (res.code !== 0) throw new Error(res.message || '上传失败')
      const url = res.data?.url
      if (!url) throw new Error('上传未返回地址')
      if (onChange) onChange(url)
    } catch (e) {
      alert(e?.data?.message || e?.message || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="admin-field">
      {label && <label>{label}</label>}
      <div className="upload-field">
        <input
          className="admin-input"
          style={{ flex: 1 }}
          value={value || ''}
          onChange={(e) => onChange && onChange(e.target.value)}
          placeholder={placeholder}
        />
        <input type="file" accept={accept} style={{ display: 'none' }} ref={fileInputRef}
          onChange={(e) => { handleFile(e.target.files[0]); e.target.value = '' }} />
        <button className="admin-btn ghost sm" disabled={uploading}
          onClick={() => fileInputRef.current?.click()}>
          {uploading ? '上传中…' : <><i className="fa-solid fa-cloud-arrow-up" /> {uploadText}</>}
        </button>
      </div>
    </div>
  )
}
