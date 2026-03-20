import { useEffect, useState } from 'react'
import { Table, Button, Space, Typography, Empty, App, Popconfirm, Grid } from 'antd'
import type { Breakpoint } from 'antd'
import { UndoOutlined, DeleteOutlined } from '@ant-design/icons'
import type { FileItem } from '@/types'
import { listTrash, restoreFile, permanentDelete } from '@/api/files'
import { formatBytes, formatDate } from '@/utils/format'

const { Title, Text } = Typography
const TABLE_MD: Breakpoint[] = ['md']
const TABLE_LG: Breakpoint[] = ['lg']

export default function TrashPage() {
    const [files, setFiles] = useState<FileItem[]>([])
    const [loading, setLoading] = useState(true)
    const { message, modal } = App.useApp()
    const screens = Grid.useBreakpoint()
    const isMobile = !screens.md

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
            responsive: TABLE_MD,
            render: (size: number, record: FileItem) => record.is_directory ? '-' : formatBytes(size),
        },
        {
            title: '删除时间',
            dataIndex: 'deleted_at',
            key: 'deleted_at',
            width: 160,
            responsive: TABLE_LG,
            render: (date: string) => date ? formatDate(date) : '-',
        },
        {
            title: '操作',
            key: 'actions',
            width: isMobile ? 108 : 160,
            render: (_: unknown, record: FileItem) => (
                <Space size={4}>
                    <Button type="link" size="small" icon={<UndoOutlined />} onClick={() => handleRestore(record.id)}>
                        {!isMobile && '恢复'}
                    </Button>
                    <Popconfirm
                        title="永久删除此文件？"
                        description="此操作不可撤销"
                        onConfirm={() => handlePermanentDelete(record.id)}
                        okText="删除"
                        cancelText="取消"
                    >
                        <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                            {!isMobile && '永久删除'}
                        </Button>
                    </Popconfirm>
                </Space>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <div className="page-header-bar">
                <Title level={4} style={{ margin: 0 }}>回收站</Title>
                {files.length > 0 && (
                    <Button danger icon={<DeleteOutlined />} onClick={handleEmptyTrash}>
                        清空回收站
                    </Button>
                )}
            </div>

            <div className="page-card">
                <Table
                    dataSource={files}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    size={isMobile ? 'small' : 'middle'}
                    scroll={isMobile ? { x: 560 } : undefined}
                    locale={{ emptyText: <Empty description="回收站为空" /> }}
                />
            </div>

            <Text type="secondary" style={{ display: 'block', marginTop: 12, fontSize: 12 }}>
                回收站中的文件将在 30 天后自动永久删除
            </Text>
        </div>
    )
}
