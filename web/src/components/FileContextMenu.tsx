import { useEffect, useRef } from 'react'
import { Menu, Input, App } from 'antd'
import {
    EyeOutlined,
    EditOutlined,
    CopyOutlined,
    ScissorOutlined,
    DeleteOutlined,
    ShareAltOutlined,
    LinkOutlined,
    FolderOpenOutlined,
    SelectOutlined,
} from '@ant-design/icons'
import type { FileItem } from '@/types'
import { useFileStore } from '@/stores/fileStore'
import { getFileURL } from '@/api/files'
import { copyToClipboard } from '@/utils/format'
import { useState } from 'react'

interface FileContextMenuProps {
    x: number
    y: number
    file: FileItem | null
    onClose: () => void
    onPreview: (file: FileItem) => void
    onShare: (fileId: string) => void
    onSelectAll: () => void
}

export default function FileContextMenu({
    x, y, file, onClose, onPreview, onShare, onSelectAll,
}: FileContextMenuProps) {
    const ref = useRef<HTMLDivElement>(null)
    const { navigateTo, renameFile, setClipboard, deleteSelected, toggleSelect } = useFileStore()
    const { message, modal } = App.useApp()
    const [renaming, setRenaming] = useState(false)
    const [renameName, setRenameName] = useState('')

    useEffect(() => {
        const handleClick = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) {
                onClose()
            }
        }
        document.addEventListener('mousedown', handleClick)
        return () => document.removeEventListener('mousedown', handleClick)
    }, [onClose])

    // Adjust position to stay within viewport
    const adjustedX = Math.min(x, window.innerWidth - 200)
    const adjustedY = Math.min(y, window.innerHeight - 300)

    const handleRename = async () => {
        if (!file || !renameName.trim()) return
        try {
            await renameFile(file.id, renameName.trim())
            message.success('重命名成功')
            setRenaming(false)
            onClose()
        } catch {
            message.error('重命名失败')
        }
    }

    const handleCopyLink = async () => {
        if (!file) return
        try {
            const { url } = await getFileURL(file.id)
            await copyToClipboard(url)
            message.success('直链已复制')
        } catch {
            message.error('获取链接失败')
        }
        onClose()
    }

    const handleDelete = () => {
        if (!file) return
        toggleSelect(file.id)
        modal.confirm({
            title: '确认删除',
            content: `确定要将 "${file.name}" 移入回收站吗？`,
            okText: '删除',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                await deleteSelected()
                message.success('已移入回收站')
            },
        })
        onClose()
    }

    if (renaming && file) {
        return (
            <div ref={ref} className="file-context-menu" style={{ left: adjustedX, top: adjustedY }}>
                <div style={{ padding: 8, background: 'var(--color-bg-elevated)', borderRadius: 8, boxShadow: '0 4px 16px var(--color-shadow)', border: '1px solid var(--color-border-secondary)' }}>
                    <Input
                        value={renameName}
                        onChange={(e) => setRenameName(e.target.value)}
                        onPressEnter={handleRename}
                        autoFocus
                        size="small"
                        placeholder="输入新名称"
                        addonAfter={<a onClick={handleRename}>确定</a>}
                    />
                </div>
            </div>
        )
    }

    // Build menu items based on context
    const items = []

    if (file) {
        if (file.is_directory) {
            items.push({ key: 'open', icon: <FolderOpenOutlined />, label: '打开', onClick: () => { navigateTo(file.id, file.name); onClose() } })
        } else {
            items.push({ key: 'preview', icon: <EyeOutlined />, label: '预览', onClick: () => { onPreview(file); onClose() } })
            items.push({ key: 'link', icon: <LinkOutlined />, label: '复制直链', onClick: handleCopyLink })
            items.push({ key: 'share', icon: <ShareAltOutlined />, label: '分享', onClick: () => { onShare(file.id); onClose() } })
        }
        items.push({ type: 'divider' as const })
        items.push({ key: 'rename', icon: <EditOutlined />, label: '重命名', onClick: () => { setRenameName(file.name); setRenaming(true) } })
        items.push({ key: 'copy', icon: <CopyOutlined />, label: '复制', onClick: () => { setClipboard([file.id], 'copy'); message.info('已复制'); onClose() } })
        items.push({ key: 'cut', icon: <ScissorOutlined />, label: '剪切', onClick: () => { setClipboard([file.id], 'cut'); message.info('已剪切'); onClose() } })
        items.push({ type: 'divider' as const })
        items.push({ key: 'delete', icon: <DeleteOutlined />, label: '删除', danger: true, onClick: handleDelete })
    } else {
        // Background context menu (no file selected)
        items.push({ key: 'select-all', icon: <SelectOutlined />, label: '全选', onClick: () => { onSelectAll(); onClose() } })
    }

    return (
        <div ref={ref} className="file-context-menu" style={{ left: adjustedX, top: adjustedY }}>
            <Menu
                items={items}
                style={{
                    borderRadius: 8,
                    boxShadow: '0 4px 16px var(--color-shadow)',
                    border: '1px solid var(--color-border-secondary)',
                    minWidth: 160,
                }}
            />
        </div>
    )
}
