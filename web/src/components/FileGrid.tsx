import { Spin, Empty, Tag } from 'antd'
import {
    FolderFilled,
    FileImageOutlined,
    FileTextOutlined,
    SoundOutlined,
    VideoCameraOutlined,
    FileOutlined,
} from '@ant-design/icons'
import type { FileItem } from '@/types'
import { useFileStore } from '@/stores/fileStore'
import { isImage, isAudio, isVideo, isText } from '@/utils/format'

interface FileGridProps {
    files: FileItem[]
    loading: boolean
    selectedIds: Set<string>
    onContextMenu: (e: React.MouseEvent, file: FileItem) => void
    onPreview: (file: FileItem) => void
}

function getGridIcon(file: FileItem) {
    if (file.is_directory) return <FolderFilled style={{ color: '#faad14' }} />
    if (isImage(file.mime_type)) return <FileImageOutlined style={{ color: '#1677ff' }} />
    if (isAudio(file.mime_type)) return <SoundOutlined style={{ color: '#722ed1' }} />
    if (isVideo(file.mime_type)) return <VideoCameraOutlined style={{ color: '#eb2f96' }} />
    if (isText(file.mime_type)) return <FileTextOutlined style={{ color: '#52c41a' }} />
    return <FileOutlined style={{ color: '#8c8c8c' }} />
}

export default function FileGrid({ files, loading, selectedIds, onContextMenu, onPreview }: FileGridProps) {
    const { toggleSelect, navigateTo } = useFileStore()

    const handleClick = (file: FileItem, e: React.MouseEvent) => {
        if (e.ctrlKey || e.metaKey) {
            toggleSelect(file.id)
            return
        }
        if (file.is_directory) {
            navigateTo(file.id, file.name)
        } else {
            onPreview(file)
        }
    }

    if (loading) {
        return (
            <div className="file-list-container" style={{ padding: 80, textAlign: 'center' }}>
                <Spin size="large" />
            </div>
        )
    }

    if (files.length === 0) {
        return (
            <div className="file-list-container empty-state">
                <Empty description="此文件夹为空" />
            </div>
        )
    }

    return (
        <div className="file-list-container">
            <div className="file-grid">
                {files.map((file) => (
                    <div
                        key={file.id}
                        className={`file-grid-item ${selectedIds.has(file.id) ? 'selected' : ''} ${file.moderation_status === 'banned' ? 'banned' : ''}`}
                        onClick={(e) => handleClick(file, e)}
                        onContextMenu={(e) => { e.stopPropagation(); onContextMenu(e, file) }}
                    >
                        <div className="file-grid-icon">{getGridIcon(file)}</div>
                        <div className="file-grid-name">{file.name}</div>
                        {file.moderation_status === 'banned' && !file.is_directory && (
                            <Tag color="red" style={{ margin: 0 }}>
                                已封禁
                            </Tag>
                        )}
                    </div>
                ))}
            </div>
        </div>
    )
}
