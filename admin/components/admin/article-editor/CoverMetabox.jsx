'use client'

import Metabox from './Metabox'
import { UploadField } from './fields'

// 封面图元框：URL 输入 + 上传 + 预览 + 移除。
export default function CoverMetabox({ form, onChange }) {
  return (
    <Metabox title="封面图" icon="fa-image">
      <UploadField
        label=""
        accept="image/*"
        type="cover"
        value={form.cover}
        onChange={(v) => onChange({ cover: v })}
        placeholder="粘贴封面图 URL 或点击上传"
        uploadText="上传封面"
      />
      {form.cover ? (
        <div className="admin-cover-preview">
          <img src={form.cover} alt="封面预览" />
          <button className="admin-btn ghost sm" type="button"
            onClick={() => onChange({ cover: '' })}>移除封面</button>
        </div>
      ) : null}
    </Metabox>
  )
}
