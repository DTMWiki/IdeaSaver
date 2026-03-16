import { useEffect, useState } from 'react'
import { Table, Button, Space, Typography, Empty, Spin, App, Popconfirm } from 'antd'
import { UndoOutlined, DeleteOutlined } from '@ant-design/icons'
import type { FileItem } from '@/types'
import { listTrash, restoreFile, permanentDelete } from '@/api/files'
import { formatBytes, formatDate } from '@/utils/format'

const { Title, Text } = Typography

export default function TrashPage() {
    const [files, setFiles] = useState<FileItem[]>([])
    const [loading, setLoading] = useState(true)
    const [selectedIds, setSelectedIds] = useState<string[]>([])
    const { message, modal } = App.useApp()

    const fetchTrash = async () => {
        setLoading(true)
        try {
            const data = await listTrash()
            setFiles(data)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchTrash() }, [])

    const handleRestore = async (id: string) => {
        await restoreFile(id)
        message.success('已恢复')
        fetchTrash()
    }

    const handlePermanentDelete = async (id: string) => {
        await permanentDelete(id)
        message.success('已永久删除')
        fetchTrash()
    }

    const handleEmptyTrash = () => {
        modal.confirm({
            title: '清空回收站',
            content: '确定要永久删除回收站中的所有文件吗？此操作不可撤销。',
            okText: '全部删除',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                for (const f of files) {
                    await permanentDelete(f.id)
                }
                message.success('回收站已清空')
                fetchTrash()
            },
        })
    }

    const columns = [
        {
            title: '文件名',
            dataIndex: 'name',
            key: 'name',
            ellipsis: true,
        },
        {
            title: '大小',
            dataIndex: 'size',
            key: 'size',
            width: 100,
            render: (size: number, record: FileItem) => record.is_directory ? '-' : formatBytes(size),
        },
        {
            title: '删除时间',
            dataIndex: 'deleted_at',
            key: 'deleted_at',
            width: 160,
            render: (date: string) => date ? formatDate(date) : '-',
        },
        {
            title: '操作',
            key: 'actions',
            width: 160,
            render: (_: unknown, record: FileItem) => (
                <Space size={4}>
                    <Button type="link" size="small" icon={<UndoOutlined />} onClick={() => handleRestore(record.id)}>
                        恢复
                    </Button>
                    <Popconfirm
                        title="永久删除此文件？"
                        description="此操作不可撤销"
                        onConfirm={() => handlePermanentDelete(record.id)}
                        okText="删除"
                        cancelText="取消"
                    >
                        <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                            永久删除
                        </Button>
                    </Popconfirm>
                </Space>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>回收站</Title>
                {files.length > 0 && (
                    <Button danger icon={<DeleteOutlined />} onClick={handleEmptyTrash}>
                        清空回收站
                    </Button>
                )}
            </div>

            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table
                    dataSource={files}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    locale={{ emptyText: <Empty description="回收站为空" /> }}
                />
            </div>

            <Text type="secondary" style={{ display: 'block', marginTop: 12, fontSize: 12 }}>
                回收站中的文件将在 30 天后自动永久删除
            </Text>
        </div>
    )
}
