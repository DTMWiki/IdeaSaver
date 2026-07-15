import { useCallback, useState } from 'react'
import { App } from 'antd'
import type { FileItem } from '@/types'
import { useFileStore } from '@/stores/fileStore'
import { getFileURL, submitFileAppeal } from '@/api/files'
import { copyToClipboard } from '@/utils/format'
import { buildFileActionItems } from '@/components/fileActionItems'

export function useDashboardActions() {
  const {
    selectedIds,
    renameFile,
    deleteFiles,
    setClipboard,
    navigateTo,
  } = useFileStore()
  const { message, modal } = App.useApp()

  const [previewFile, setPreviewFile] = useState<FileItem | null>(null)
  const [shareFileId, setShareFileId] = useState<string | null>(null)
  const [appealFile, setAppealFile] = useState<FileItem | null>(null)
  const [appealReason, setAppealReason] = useState('')
  const [appealSubmitting, setAppealSubmitting] = useState(false)
  const [renameTarget, setRenameTarget] = useState<FileItem | null>(null)
  const [renameName, setRenameName] = useState('')
  const [renameSubmitting, setRenameSubmitting] = useState(false)

  const resolveTargetIDs = useCallback((file: FileItem) => {
    const currentSelection = useFileStore.getState().selectedIds
    if (currentSelection.has(file.id) && currentSelection.size > 1) {
      return Array.from(currentSelection)
    }
    return [file.id]
  }, [])

  const openRename = useCallback((file: FileItem) => {
    setRenameTarget(file)
    setRenameName(file.name)
  }, [])

  const handleConfirmRename = useCallback(async () => {
    if (!renameTarget || !renameName.trim()) return
    setRenameSubmitting(true)
    try {
      await renameFile(renameTarget.id, renameName.trim())
      message.success('重命名成功')
      setRenameTarget(null)
      setRenameName('')
    } catch {
      message.error('重命名失败')
    } finally {
      setRenameSubmitting(false)
    }
  }, [message, renameFile, renameName, renameTarget])

  const handleCopyLink = useCallback(async (file: FileItem) => {
    try {
      const { url } = await getFileURL(file.id)
      await copyToClipboard(url)
      message.success('直链已复制')
    } catch {
      message.error('获取链接失败')
    }
  }, [message])

  const handleCopyMarkdown = useCallback(async (file: FileItem) => {
    try {
      const { markdown, url } = await getFileURL(file.id)
      await copyToClipboard(markdown || `[${file.name}](${url})`)
      message.success('Markdown 已复制')
    } catch {
      message.error('获取 Markdown 失败')
    }
  }, [message])

  const handleDelete = useCallback((file: FileItem) => {
    const ids = resolveTargetIDs(file)
    const batch = ids.length > 1
    modal.confirm({
      title: batch ? '确认批量删除' : '确认删除',
      content: batch
        ? `确定要将 ${ids.length} 个项目移入回收站吗？`
        : `确定要将 "${file.name}" 移入回收站吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deleteFiles(ids)
        message.success(
          batch ? `已移入回收站 (${ids.length} 项)` : '已移入回收站',
        )
      },
    })
  }, [deleteFiles, message, modal, resolveTargetIDs])

  const handleSubmitAppeal = useCallback(async () => {
    if (!appealFile) return
    const reason = appealReason.trim()
    if (!reason) {
      message.warning('请填写申诉理由')
      return
    }

    setAppealSubmitting(true)
    try {
      await submitFileAppeal(appealFile.id, reason)
      message.success('申诉工单已提交，请等待管理员审核')
      setAppealFile(null)
      setAppealReason('')
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error
      message.error(maybeMessage || '提交申诉失败')
    } finally {
      setAppealSubmitting(false)
    }
  }, [appealFile, appealReason, message])

  const fileActionItems = useCallback((file: FileItem) =>
    buildFileActionItems(file, {
      onOpen: (target) => navigateTo(target.id, target.name),
      onPreview: (target) => setPreviewFile(target),
      onCopyLink: handleCopyLink,
      onCopyMarkdown: handleCopyMarkdown,
      onShare: (target) => setShareFileId(target.id),
      onAppeal: (target) => setAppealFile(target),
      onRename: openRename,
      onCopy: (target) => {
        const ids = resolveTargetIDs(target)
        setClipboard(ids, 'copy')
        message.info(
          ids.length > 1
            ? `已复制 ${ids.length} 个项目`
            : `已复制 "${target.name}"`,
        )
      },
      onCut: (target) => {
        const ids = resolveTargetIDs(target)
        setClipboard(ids, 'cut')
        message.info(
          ids.length > 1
            ? `已剪切 ${ids.length} 个项目`
            : `已剪切 "${target.name}"`,
        )
      },
      onDelete: handleDelete,
    }), [
    handleCopyLink,
    handleCopyMarkdown,
    handleDelete,
    message,
    navigateTo,
    openRename,
    resolveTargetIDs,
    setClipboard,
  ])

  return {
    selectedIds,
    previewFile,
    setPreviewFile,
    shareFileId,
    setShareFileId,
    appealFile,
    setAppealFile,
    appealReason,
    setAppealReason,
    appealSubmitting,
    renameTarget,
    setRenameTarget,
    renameName,
    setRenameName,
    renameSubmitting,
    openRename,
    handleConfirmRename,
    handleSubmitAppeal,
    fileActionItems,
    resolveTargetIDs,
  }
}
