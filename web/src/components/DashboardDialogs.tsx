import { Input, Modal } from 'antd'
import type { FileItem } from '@/types'
import FilePreview from '@/components/FilePreview'
import CreateShareModal from '@/components/CreateShareModal'

interface DashboardDialogsProps {
  mkdirVisible: boolean
  mkdirName: string
  onMkdirNameChange: (value: string) => void
  onMkdirOk: () => void
  onMkdirCancel: () => void
  renameTarget: FileItem | null
  renameName: string
  renameSubmitting: boolean
  onRenameNameChange: (value: string) => void
  onRenameOk: () => void
  onRenameCancel: () => void
  previewFile: FileItem | null
  onPreviewClose: () => void
  shareFileId: string | null
  onShareClose: () => void
  appealFile: FileItem | null
  appealReason: string
  appealSubmitting: boolean
  onAppealReasonChange: (value: string) => void
  onAppealOk: () => void
  onAppealCancel: () => void
}

export default function DashboardDialogs({
  mkdirVisible,
  mkdirName,
  onMkdirNameChange,
  onMkdirOk,
  onMkdirCancel,
  renameTarget,
  renameName,
  renameSubmitting,
  onRenameNameChange,
  onRenameOk,
  onRenameCancel,
  previewFile,
  onPreviewClose,
  shareFileId,
  onShareClose,
  appealFile,
  appealReason,
  appealSubmitting,
  onAppealReasonChange,
  onAppealOk,
  onAppealCancel,
}: DashboardDialogsProps) {
  return (
    <>
      <Modal
        title="新建文件夹"
        open={mkdirVisible}
        onOk={onMkdirOk}
        onCancel={onMkdirCancel}
        okText="创建"
        cancelText="取消"
      >
        <Input
          placeholder="文件夹名称"
          value={mkdirName}
          onChange={(e) => onMkdirNameChange(e.target.value)}
          onPressEnter={onMkdirOk}
          autoFocus
        />
      </Modal>

      <Modal
        title="重命名"
        open={!!renameTarget}
        onOk={onRenameOk}
        onCancel={onRenameCancel}
        okText="保存"
        cancelText="取消"
        confirmLoading={renameSubmitting}
      >
        <Input
          placeholder="输入新名称"
          value={renameName}
          onChange={(e) => onRenameNameChange(e.target.value)}
          onPressEnter={onRenameOk}
          autoFocus
        />
      </Modal>

      <FilePreview file={previewFile} onClose={onPreviewClose} />
      <CreateShareModal fileId={shareFileId} onClose={onShareClose} />

      <Modal
        title="提交申诉"
        open={!!appealFile}
        onOk={onAppealOk}
        onCancel={onAppealCancel}
        okText="提交"
        cancelText="取消"
        confirmLoading={appealSubmitting}
      >
        <Input.TextArea
          rows={4}
          maxLength={1000}
          showCount
          placeholder="请说明申诉理由"
          value={appealReason}
          onChange={(e) => onAppealReasonChange(e.target.value)}
        />
      </Modal>
    </>
  )
}
