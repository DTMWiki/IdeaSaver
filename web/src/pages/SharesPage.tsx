import { useEffect, useState } from 'react'
import { Table, Button, Typography, Empty, Tag, Popconfirm, App } from 'antd'
import { DeleteOutlined, CopyOutlined } from '@ant-design/icons'
import type { Share } from '@/types'
import { listShares, deleteShare } from '@/api/shares'
import { formatDate, copyToClipboard } from '@/utils/format'

const { Title } = Typography

export default function SharesPage() {
    const [shares, setShares] = useState<Share[]>([])
    const [loading, setLoading] = useState(true)
    const { message } = App.useApp()

    const fetchShares = async () => {
        setLoading(true)
        try {
            const data = await listShares()
            setShares(data)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchShares() }, [])

    const handleDelete = async (id: string) => {
        await deleteShare(id)
        message.success('分享已删除')
        fetchShares()
    }

    const handleCopyLink = async (code: string) => {
        const url = `${window.location.origin}/share/${code}`
        await copyToClipboard(url)
        message.success('链接已复制')
    }

    const columns = [
        {
            title: '分享码',
            dataIndex: 'code',
            key: 'code',
            width: 140,
            render: (code: string) => (
                <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => handleCopyLink(code)}>
                    {code}
                </Button>
            ),
        },
        {
            title: '文件',
            dataIndex: 'file_name',
            key: 'file_name',
            ellipsis: true,
            render: (name: string) => name || '-',
        },
        {
            title: '查看次数',
            dataIndex: 'view_count',
            key: 'view_count',
            width: 100,
        },
        {
            title: '过期时间',
            dataIndex: 'expires_at',
            key: 'expires_at',
            width: 180,
            render: (date: string | null) => {
                if (!date) return <Tag color="blue">永久</Tag>
                const expired = new Date(date) < new Date()
                return expired ? <Tag color="red">已过期</Tag> : formatDate(date)
            },
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
            render: (_: unknown, record: Share) => (
                <Popconfirm
                    title="删除此分享链接？"
                    onConfirm={() => handleDelete(record.id)}
                    okText="删除"
                    cancelText="取消"
                >
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <Title level={4} style={{ marginBottom: 16 }}>我的分享</Title>

            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table
                    dataSource={shares}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    locale={{ emptyText: <Empty description="暂无分享链接" /> }}
                />
            </div>
        </div>
    )
}
