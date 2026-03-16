import { Table, Checkbox, Space, Typography, Button, Tooltip } from 'antd'
import {
    FolderFilled,
    FileImageOutlined,
    FileTextOutlined,
    SoundOutlined,
    VideoCameraOutlined,
    FileOutlined,
    EyeOutlined,
    LinkOutlined,
    ShareAltOutlined,
} from '@ant-design/icons'
import { App } from 'antd'
import type { FileItem } from '@/types'
import { useFileStore } from '@/stores/fileStore'
import { formatBytes, formatDate, isImage, isAudio, isVideo, isText, copyToClipboard } from '@/utils/format'
import { getFileURL } from '@/api/files'

const { Text } = Typography

interface FileTableProps {
    files: FileItem[]
    loading: boolean
    selectedIds: Set<string>
    onContextMenu: (e: React.MouseEvent, file: FileItem) => void
    onPreview: (file: FileItem) => void
    onShare: (fileId: string) => void
}

function getFileIcon(file: FileItem) {
    if (file.is_directory) return <FolderFilled style={{ color: '#faad14', fontSize: 18 }} />
    if (isImage(file.mime_type)) return <FileImageOutlined style={{ color: '#1677ff', fontSize: 18 }} />
    if (isAudio(file.mime_type)) return <SoundOutlined style={{ color: '#722ed1', fontSize: 18 }} />
    if (isVideo(file.mime_type)) return <VideoCameraOutlined style={{ color: '#eb2f96', fontSize: 18 }} />
    if (isText(file.mime_type)) return <FileTextOutlined style={{ color: '#52c41a', fontSize: 18 }} />
    return <FileOutlined style={{ color: '#8c8c8c', fontSize: 18 }} />
}

export default function FileTable({ files, loading, selectedIds, onContextMenu, onPreview, onShare }: FileTableProps) {
    const { toggleSelect, navigateTo } = useFileStore()
    const { message } = App.useApp()

    const handleRowClick = (file: FileItem) => {
        if (file.is_directory) {
            navigateTo(file.id, file.name)
        } else {
            onPreview(file)
        }
    }

    const handleCopyLink = async (file: FileItem) => {
        try {
            const { url } = await getFileURL(file.id)
            await copyToClipboard(url)
            message.success('直链已复制')
        } catch {
            message.error('获取链接失败')
        }
    }

    const columns = [
        {
            title: '',
            dataIndex: 'id',
            key: 'select',
            width: 40,
            render: (_: string, record: FileItem) => (
                <Checkbox
                    checked={selectedIds.has(record.id)}
                    onChange={() => toggleSelect(record.id)}
                    onClick={(e) => e.stopPropagation()}
                />
            ),
        },
        {
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            ellipsis: true,
            render: (_: string, record: FileItem) => (
                <Space>
                    {getFileIcon(record)}
                    <Text ellipsis={{ tooltip: record.name }}>{record.name}</Text>
                </Space>
            ),
        },
        {
            title: '大小',
            dataIndex: 'size',
            key: 'size',
            width: 100,
            render: (size: number, record: FileItem) =>
                record.is_directory ? '-' : formatBytes(size),
        },
        {
            title: '修改时间',
            dataIndex: 'updated_at',
            key: 'updated_at',
            width: 160,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: 120,
            render: (_: unknown, record: FileItem) =>
                record.is_directory ? null : (
                    <Space size={4}>
                        <Tooltip title="预览">
                            <Button type="text" size="small" icon={<EyeOutlined />} onClick={(e) => { e.stopPropagation(); onPreview(record) }} />
                        </Tooltip>
                        <Tooltip title="直链">
                            <Button type="text" size="small" icon={<LinkOutlined />} onClick={(e) => { e.stopPropagation(); handleCopyLink(record) }} />
                        </Tooltip>
                        <Tooltip title="分享">
                            <Button type="text" size="small" icon={<ShareAltOutlined />} onClick={(e) => { e.stopPropagation(); onShare(record.id) }} />
                        </Tooltip>
                    </Space>
                ),
        },
    ]

    return (
        <div className="file-list-container">
            <Table
                dataSource={files}
                columns={columns}
                rowKey="id"
                loading={loading}
                pagination={false}
                size="middle"
                locale={{ emptyText: '此文件夹为空' }}
                onRow={(record) => ({
                    onClick: () => handleRowClick(record),
                    onContextMenu: (e) => onContextMenu(e, record),
                    className: 'file-table-row',
                })}
            />
        </div>
    )
}
