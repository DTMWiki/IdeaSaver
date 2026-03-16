import { useEffect, useState } from 'react'
import { Table, Button, Typography, Empty, Popconfirm, App, Pagination, Tag, Space, Modal, Input, Tooltip } from 'antd'
import { DeleteOutlined, StopOutlined, CheckCircleOutlined } from '@ant-design/icons'
import type { FileItem } from '@/types'
import { listAllFiles, adminDeleteFile, adminBanFile, adminUnbanFile } from '@/api/admin'
import { formatBytes, formatDate } from '@/utils/format'

const { Title } = Typography

export default function AdminFiles() {
    const [files, setFiles] = useState<FileItem[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const [banTarget, setBanTarget] = useState<FileItem | null>(null)
    const [banReason, setBanReason] = useState('')
    const [actionLoading, setActionLoading] = useState(false)
    const pageSize = 50
    const { message } = App.useApp()

    const fetchFiles = async (p: number) => {
        setLoading(true)
        try {
            const data = await listAllFiles((p - 1) * pageSize, pageSize)
            setFiles(data.files)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchFiles(1) }, [])

    const handleDelete = async (id: string) => {
        await adminDeleteFile(id)
        message.success('已删除')
        fetchFiles(page)
    }

    const handleBan = async () => {
        if (!banTarget) return
        const reason = banReason.trim()
        if (!reason) {
            message.warning('请输入封禁理由')
            return
        }
        setActionLoading(true)
        try {
            await adminBanFile(banTarget.id, reason)
            message.success('文件已封禁')
            setBanTarget(null)
            setBanReason('')
            fetchFiles(page)
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '封禁失败')
        } finally {
            setActionLoading(false)
        }
    }

    const handleUnban = async (record: FileItem) => {
        setActionLoading(true)
        try {
            await adminUnbanFile(record.id)
            message.success('已解除封禁')
            fetchFiles(page)
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '解除失败')
        } finally {
            setActionLoading(false)
        }
    }

    const columns = [
        {
            title: '文件名',
            dataIndex: 'name',
            key: 'name',
            ellipsis: true,
        },
        {
            title: '用户',
            dataIndex: 'user_id',
            key: 'user_id',
            width: 120,
            ellipsis: true,
            render: (id: string) => id.slice(0, 8) + '...',
        },
        {
            title: '类型',
            dataIndex: 'is_directory',
            key: 'type',
            width: 80,
            render: (isDir: boolean) => isDir ? '文件夹' : '文件',
        },
        {
            title: '状态',
            dataIndex: 'moderation_status',
            key: 'moderation_status',
            width: 100,
            render: (status: FileItem['moderation_status'], record: FileItem) => {
                if (record.is_directory) return '-'
                return status === 'banned' ? <Tag color="red">已封禁</Tag> : <Tag color="green">正常</Tag>
            },
        },
        {
            title: '大小',
            dataIndex: 'size',
            key: 'size',
            width: 100,
            render: (size: number, record: FileItem) => record.is_directory ? '-' : formatBytes(size),
        },
        {
            title: '封禁原因',
            dataIndex: 'moderation_reason',
            key: 'moderation_reason',
            ellipsis: true,
            render: (reason?: string) => reason ? <Tooltip title={reason}>{reason}</Tooltip> : '-',
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
            width: 180,
            render: (_: unknown, record: FileItem) => (
                <Space size={4}>
                    {!record.is_directory && (
                        record.moderation_status === 'banned' ? (
                            <Button type="text" size="small" icon={<CheckCircleOutlined />} onClick={() => handleUnban(record)} loading={actionLoading}>
                                解封
                            </Button>
                        ) : (
                            <Button type="text" size="small" danger icon={<StopOutlined />} onClick={() => setBanTarget(record)}>
                                封禁
                            </Button>
                        )
                    )}
                    <Popconfirm title="永久删除此文件？" onConfirm={() => handleDelete(record.id)} okText="删除" cancelText="取消">
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                </Space>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <Title level={4} style={{ marginBottom: 16 }}>全局文件管理</Title>
            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table
                    dataSource={files}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    locale={{ emptyText: <Empty description="暂无文件" /> }}
                />
            </div>
            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={fetchFiles} showTotal={(t) => `共 ${t} 个文件`} />
                </div>
            )}

            <Modal
                title={banTarget ? `封禁文件：${banTarget.name}` : '封禁文件'}
                open={!!banTarget}
                onOk={handleBan}
                onCancel={() => {
                    setBanTarget(null)
                    setBanReason('')
                }}
                okText="确认封禁"
                cancelText="取消"
                confirmLoading={actionLoading}
                okButtonProps={{ danger: true }}
                destroyOnClose
            >
                <Input.TextArea
                    value={banReason}
                    onChange={(e) => setBanReason(e.target.value)}
                    placeholder="请输入封禁原因（将展示给用户）"
                    rows={4}
                    maxLength={1000}
                    showCount
                />
            </Modal>
        </div>
    )
}
