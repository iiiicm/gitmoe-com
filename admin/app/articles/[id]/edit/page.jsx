'use client'

// 编辑文章（整页编辑器薄壳）。
// 注意：本页不使用 useSearchParams，因此无需 <Suspense> 包裹；
// 若将来引入 useSearchParams，next build 会强制要求包 <Suspense>。
import ArticleEditorScreen from '../../../../components/admin/article-editor/ArticleEditorScreen'

export default function EditArticlePage({ params }) {
  const id = params?.id ? Number(params.id) : null
  return <ArticleEditorScreen mode="edit" articleId={Number.isFinite(id) ? id : null} />
}
