import { useEffect, useState } from 'react'
import { Table, Button, Typography, Empty, Popconfirm, App, Pagination, Tag } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { Video } from '@/types'
import { listAllVideos, adminDeleteVideo } from '@/api/admin'
import { formatBytes, formatDate } from '@/utils/format'

const { Title } = Typography

export default function AdminVideos() {
    const [videos, setVideos] = useState<Video[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const pageSize = 50
    const { message } = App.useApp()

    const fetchVideos = async (p: number) => {
        setLoading(true)
        try {
            const data = await listAllVideos((p - 1) * pageSize, pageSize)
            setVideos(data.videos)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchVideos(1) }, [])

    const handleDelete = async (id: string) => {
        await adminDeleteVideo(id)
        message.success('已删除')
        fetchVideos(page)
    }

    const columns = [
        { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
        {
            title: '用户',
            dataIndex: 'user_id',
            key: 'user_id',
            width: 120,
            ellipsis: true,
            render: (id: string) => id.slice(0, 8) + '...',
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 80,
            render: (s: number) => s === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">禁用</Tag>,
        },
        {
            title: '大小',
            dataIndex: 'size',
            key: 'size',
            width: 100,
            render: (size: number) => formatBytes(size),
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: 80,
            render: (_: unknown, record: Video) => (
                <Popconfirm title="删除此视频？" onConfirm={() => handleDelete(record.id)} okText="删除" cancelText="取消">
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <Title level={4} style={{ marginBottom: 16 }}>全局视频管理</Title>
            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table dataSource={videos} columns={columns} rowKey="id" loading={loading} pagination={false} locale={{ emptyText: <Empty description="暂无视频" /> }} />
            </div>
            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={fetchVideos} showTotal={(t) => `共 ${t} 个视频`} />
                </div>
            )}
        </div>
    )
}
