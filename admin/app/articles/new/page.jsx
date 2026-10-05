'use client'

// 新建文章（整页编辑器薄壳）。静态段 new 的优先级高于动态段 [id]，二者共存无冲突。
import ArticleEditorScreen from '../../../components/admin/article-editor/ArticleEditorScreen'

export default function NewArticlePage() {
  return <ArticleEditorScreen mode="create" />
}
