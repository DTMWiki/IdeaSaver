import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Table, Button, Typography, Empty, Tag, Popconfirm, App, Grid, Space } from 'antd'
import type { Breakpoint } from 'antd'
import { DeleteOutlined, CopyOutlined, LinkOutlined, FolderOpenOutlined } from '@ant-design/icons'
import type { Share } from '@/types'
import { listShares, deleteShare } from '@/api/shares'
import { formatDate, copyToClipboard } from '@/utils/format'

const { Title, Text } = Typography
const TABLE_MD: Breakpoint[] = ['md']
const TABLE_LG: Breakpoint[] = ['lg']

export default function SharesPage() {
    const [shares, setShares] = useState<Share[]>([])
    const [loading, setLoading] = useState(true)
    const { message } = App.useApp()
    const screens = Grid.useBreakpoint()
    const isMobile = !screens.md
    const navigate = useNavigate()

    const fetchShares = async () => {
        setLoading(true)
        try {
            const data = await listShares()
            setShares(data)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { void fetchShares() }, [])

    const handleDelete = async (id: string) => {
        try {
            await deleteShare(id)
            message.success('分享已删除')
            void fetchShares()
        } catch {
            message.error('删除失败')
        }
    }

    const handleCopyLink = async (code: string) => {
        const url = `${window.location.origin}/share/${code}`
        await copyToClipboard(url)
        message.success('链接已复制')
    }

    const columns = [
        {
            title: '文件',
            dataIndex: 'file_name',
            key: 'file_name',
            ellipsis: true,
            render: (name: string, record: Share) => (
                <Space direction="vertical" size={0}>
                    <Text>{name || '（文件已删除）'}</Text>
                    {record.has_password && <Tag color="orange">需密码</Tag>}
                </Space>
            ),
        },
        {
            title: '链接',
            dataIndex: 'code',
            key: 'code',
            width: isMobile ? 100 : 200,
            render: (code: string) => (
                <Space size={4} wrap>
                    <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => { void handleCopyLink(code) }}>
                        复制
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        icon={<LinkOutlined />}
                        href={`/share/${code}`}
                        target="_blank"
                        rel="noreferrer"
                    >
                        打开
                    </Button>
                </Space>
            ),
        },
        {
            title: '查看',
            dataIndex: 'view_count',
            key: 'view_count',
            width: 72,
            responsive: TABLE_MD,
        },
        {
            title: '过期',
            dataIndex: 'expires_at',
            key: 'expires_at',
            width: 160,
            responsive: TABLE_LG,
            render: (date: string | null) => {
                if (!date) return <Tag color="blue">永久</Tag>
                const expired = new Date(date) < new Date()
                return expired
                    ? <Tag color="red">已过期</Tag>
                    : <Text type="secondary">{formatDate(date)}</Text>
            },
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            responsive: TABLE_LG,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: 68,
            render: (_: unknown, record: Share) => (
                <Popconfirm
                    title="删除此分享链接？"
                    description="删除后原链接将立即失效"
                    onConfirm={() => { void handleDelete(record.id) }}
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
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
                <Title level={4} style={{ margin: 0 }}>我的分享</Title>
                <Button icon={<FolderOpenOutlined />} onClick={() => navigate('/dashboard')}>
                    去文件管理创建
                </Button>
            </div>

            <div className="page-card">
                <Table
                    dataSource={shares}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    size={isMobile ? 'small' : 'middle'}
                    scroll={isMobile ? { x: 560 } : undefined}
                    locale={{
                        emptyText: (
                            <Empty description="还没有分享链接">
                                <Button type="primary" onClick={() => navigate('/dashboard')}>
                                    去文件页创建分享
                                </Button>
                            </Empty>
                        ),
                    }}
                />
            </div>
        </div>
    )
}
