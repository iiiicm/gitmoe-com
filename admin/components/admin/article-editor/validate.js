// 表单校验：通用必填 + 五类型条件必填。返回 { ok, errors }。
import { PAID_MODULES } from './constants'

export function validateForm(form) {
  const errors = {}

  if (!form.title || !form.title.trim()) {
    errors.title = '请填写标题'
  }

  // 下载模块：游戏设备为显式必填下拉（后端 autoDetectGameFields 仅在为空时推断，不会覆盖显式值）。
  if (form.contentType === 'download') {
    if (!form.gamePlatform) {
      errors.gamePlatform = '请选择游戏设备（下载模块必填）'
    }
  }

  return { ok: Object.keys(errors).length === 0, errors }
}
