import { useEffect, useRef, useState } from 'react'
import { Button, Space, Breadcrumb, Input, Modal, Segmented, Tooltip, App } from 'antd'
import {
    UploadOutlined,
    FolderAddOutlined,
    ReloadOutlined,
    AppstoreOutlined,
    BarsOutlined,
    DeleteOutlined,
    ScissorOutlined,
    CopyOutlined,
    SnippetsOutlined,
} from '@ant-design/icons'
import { useFileStore } from '@/stores/fileStore'
import { useUploadStore } from '@/stores/uploadStore'
import FileTable from '@/components/FileTable'
import FileGrid from '@/components/FileGrid'
import FileContextMenu from '@/components/FileContextMenu'
import FilePreview from '@/components/FilePreview'
import CreateShareModal from '@/components/CreateShareModal'
import type { FileItem } from '@/types'
import { submitFileAppeal } from '@/api/files'
import './Dashboard.css'

export default function Dashboard() {
    const {
        files, loading, breadcrumbs, selectedIds, viewMode, clipboardAction,
        fetchFiles, navigateToBreadcrumb, setViewMode,
        createDirectory, deleteSelected, selectAll, clearSelection,
        setClipboard, pasteFiles,
    } = useFileStore()
    const { addFiles } = useUploadStore()
    const { message, modal } = App.useApp()

    const fileInputRef = useRef<HTMLInputElement>(null)
    const [mkdirVisible, setMkdirVisible] = useState(false)
    const [mkdirName, setMkdirName] = useState('')
    const [previewFile, setPreviewFile] = useState<FileItem | null>(null)
    const [shareFileId, setShareFileId] = useState<string | null>(null)
    const [appealFile, setAppealFile] = useState<FileItem | null>(null)
    const [appealReason, setAppealReason] = useState('')
    const [appealSubmitting, setAppealSubmitting] = useState(false)

    // Context menu state
    const [contextMenu, setContextMenu] = useState<{
        x: number; y: number; file: FileItem | null
    } | null>(null)

    useEffect(() => {
        fetchFiles(null)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // ---- Handlers ----
    const handleUploadClick = () => fileInputRef.current?.click()

    const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const fileList = e.target.files
        if (!fileList || fileList.length === 0) return
        const currentParentId = useFileStore.getState().currentParentId
        addFiles(Array.from(fileList), currentParentId)
        e.target.value = '' // reset
    }

    const handleMkdir = async () => {
        if (!mkdirName.trim()) return
        try {
            await createDirectory(mkdirName.trim())
            message.success('文件夹已创建')
            setMkdirVisible(false)
            setMkdirName('')
        } catch {
            message.error('创建失败')
        }
    }

    const handleDelete = () => {
        if (selectedIds.size === 0) return
        modal.confirm({
            title: '确认删除',
            content: `确定要将 ${selectedIds.size} 个项目移入回收站吗？`,
            okText: '删除',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                await deleteSelected()
                message.success('已移入回收站')
            },
        })
    }

    const handleCopy = () => {
        const ids = Array.from(selectedIds)
        if (ids.length === 0) return
        setClipboard(ids, 'copy')
        message.info(`已复制 ${ids.length} 个项目`)
    }

    const handleCut = () => {
        const ids = Array.from(selectedIds)
        if (ids.length === 0) return
        setClipboard(ids, 'cut')
        message.info(`已剪切 ${ids.length} 个项目`)
    }

    const handlePaste = async () => {
        try {
            await pasteFiles()
            message.success('粘贴完成')
        } catch {
            message.error('粘贴失败')
        }
    }

    const handleContextMenu = (e: React.MouseEvent, file: FileItem | null) => {
        e.preventDefault()
        setContextMenu({ x: e.clientX, y: e.clientY, file })
    }

    const handleSubmitAppeal = async () => {
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
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '提交申诉失败')
        } finally {
            setAppealSubmitting(false)
        }
    }

    return (
        <div className="dashboard-page fade-in" onContextMenu={(e) => handleContextMenu(e, null)}>
            {/* Toolbar */}
            <div className="dashboard-toolbar">
                <Breadcrumb
                    items={breadcrumbs.map((b, i) => ({
                        title: (
                            <a onClick={() => navigateToBreadcrumb(i)}>{b.name}</a>
                        ),
                    }))}
                />

                <Space size={8} wrap>
                    {selectedIds.size > 0 && (
                        <>
                            <Tooltip title="复制">
                                <Button size="small" icon={<CopyOutlined />} onClick={handleCopy} />
                            </Tooltip>
                            <Tooltip title="剪切">
                                <Button size="small" icon={<ScissorOutlined />} onClick={handleCut} />
                            </Tooltip>
                            <Tooltip title="删除">
                                <Button size="small" danger icon={<DeleteOutlined />} onClick={handleDelete} />
                            </Tooltip>
                            <Button size="small" onClick={clearSelection}>
                                取消选择 ({selectedIds.size})
                            </Button>
                        </>
                    )}
                    {clipboardAction && (
                        <Tooltip title="粘贴">
                            <Button size="small" icon={<SnippetsOutlined />} onClick={handlePaste} />
                        </Tooltip>
                    )}
                    <Button icon={<FolderAddOutlined />} onClick={() => setMkdirVisible(true)}>
                        新建文件夹
                    </Button>
                    <Button type="primary" icon={<UploadOutlined />} onClick={handleUploadClick}>
                        上传文件
                    </Button>
                    <Tooltip title="刷新">
                        <Button icon={<ReloadOutlined />} onClick={() => fetchFiles()} />
                    </Tooltip>
                    <Segmented
                        size="small"
                        value={viewMode}
                        onChange={(v) => setViewMode(v as 'table' | 'grid')}
                        options={[
                            { value: 'table', icon: <BarsOutlined /> },
                            { value: 'grid', icon: <AppstoreOutlined /> },
                        ]}
                    />
                </Space>
            </div>

            {/* File List */}
            {viewMode === 'table' ? (
                <FileTable
                    files={files}
                    loading={loading}
                    selectedIds={selectedIds}
                    onContextMenu={handleContextMenu}
                    onPreview={setPreviewFile}
                    onShare={(id) => setShareFileId(id)}
                    onAppeal={setAppealFile}
                />
            ) : (
                <FileGrid
                    files={files}
                    loading={loading}
                    selectedIds={selectedIds}
                    onContextMenu={handleContextMenu}
                    onPreview={setPreviewFile}
                />
            )}

            {/* Hidden file input */}
            <input
                ref={fileInputRef}
                type="file"
                multiple
                style={{ display: 'none' }}
                onChange={handleFileSelect}
            />

            {/* Mkdir Modal */}
            <Modal
                title="新建文件夹"
                open={mkdirVisible}
                onOk={handleMkdir}
                onCancel={() => { setMkdirVisible(false); setMkdirName('') }}
                okText="创建"
                cancelText="取消"
            >
                <Input
                    placeholder="文件夹名称"
                    value={mkdirName}
                    onChange={(e) => setMkdirName(e.target.value)}
                    onPressEnter={handleMkdir}
                    autoFocus
                />
            </Modal>

            {/* Context Menu */}
            {contextMenu && (
                <FileContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    file={contextMenu.file}
                    onClose={() => setContextMenu(null)}
                    onPreview={setPreviewFile}
                    onShare={(id) => setShareFileId(id)}
                    onAppeal={setAppealFile}
                    onSelectAll={selectAll}
                />
            )}

            {/* File Preview */}
            <FilePreview file={previewFile} onClose={() => setPreviewFile(null)} />

            {/* Create Share Modal */}
            <CreateShareModal fileId={shareFileId} onClose={() => setShareFileId(null)} />

            {/* Appeal Modal */}
            <Modal
                title={appealFile ? `提交申诉：${appealFile.name}` : '提交申诉'}
                open={!!appealFile}
                onOk={handleSubmitAppeal}
                onCancel={() => {
                    setAppealFile(null)
                    setAppealReason('')
                }}
                okText="提交工单"
                cancelText="取消"
                confirmLoading={appealSubmitting}
                destroyOnClose
            >
                <Input.TextArea
                    value={appealReason}
                    onChange={(e) => setAppealReason(e.target.value)}
                    placeholder="请说明你认为该文件应恢复访问的理由（例如用途、来源、已整改内容）"
                    rows={5}
                    maxLength={1000}
                    showCount
                />
            </Modal>
        </div>
    )
}
