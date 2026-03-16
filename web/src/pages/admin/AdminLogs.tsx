import { useEffect, useState } from 'react'
import { Table, Typography, Empty, Select, Pagination, Tag, Space } from 'antd'
import type { AuditLog } from '@/types'
import { listLogs } from '@/api/admin'
import { formatDate } from '@/utils/format'

const { Title, Text } = Typography

const ACTION_OPTIONS = [
    { label: '全部', value: '' },
    { label: '登录', value: 'login' },
    { label: '上传', value: 'upload' },
    { label: '删除', value: 'delete' },
    { label: '重命名', value: 'rename' },
    { label: '移动', value: 'move' },
    { label: '复制', value: 'copy' },
    { label: '分享', value: 'share' },
    { label: '恢复', value: 'restore' },
    { label: '视频上传', value: 'video_upload' },
    { label: '视频删除', value: 'video_delete' },
    { label: '文件封禁', value: 'file_banned' },
    { label: '文件解封', value: 'file_unbanned' },
    { label: '提交申诉', value: 'file_appeal_submitted' },
    { label: '申诉通过', value: 'file_appeal_approved' },
    { label: '申诉删除', value: 'file_appeal_deleted' },
]

const ACTION_COLORS: Record<string, string> = {
    login: 'blue',
    upload: 'green',
    delete: 'red',
    rename: 'orange',
    move: 'purple',
    copy: 'cyan',
    share: 'magenta',
    restore: 'lime',
    video_upload: 'geekblue',
    video_delete: 'volcano',
    file_banned: 'red',
    file_unbanned: 'green',
    file_appeal_submitted: 'orange',
    file_appeal_approved: 'cyan',
    file_appeal_deleted: 'volcano',
}

export default function AdminLogs() {
    const [logs, setLogs] = useState<AuditLog[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const [actionFilter, setActionFilter] = useState('')
    const pageSize = 50

    const fetchLogs = async (p: number, action?: string) => {
        setLoading(true)
        try {
            const a = action !== undefined ? action : actionFilter
            const data = await listLogs(a || undefined, (p - 1) * pageSize, pageSize)
            setLogs(data.logs)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchLogs(1) }, [])

    const handleFilterChange = (action: string) => {
        setActionFilter(action)
        fetchLogs(1, action)
    }

    const columns = [
        {
            title: '操作',
            dataIndex: 'action',
            key: 'action',
            width: 120,
            render: (action: string) => <Tag color={ACTION_COLORS[action] || 'default'}>{action}</Tag>,
        },
        { title: '用户', dataIndex: 'username', key: 'username', width: 120, ellipsis: true },
        { title: '资源', dataIndex: 'resource', key: 'resource', width: 80 },
        {
            title: '详情',
            dataIndex: 'details',
            key: 'details',
            ellipsis: true,
            render: (details: Record<string, unknown>) =>
                details ? <Text type="secondary" style={{ fontSize: 12 }}>{JSON.stringify(details)}</Text> : '-',
        },
        {
            title: 'IP',
            dataIndex: 'ip_address',
            key: 'ip_address',
            width: 140,
            render: (ip: string) => ip || '-',
        },
        {
            title: '时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (date: string) => formatDate(date),
        },
    ]

    return (
        <div className="fade-in">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
                <Title level={4} style={{ margin: 0 }}>审计日志</Title>
                <Space>
                    <Text type="secondary">筛选:</Text>
                    <Select
                        value={actionFilter}
                        onChange={handleFilterChange}
                        options={ACTION_OPTIONS}
                        style={{ width: 140 }}
                        size="small"
                    />
                </Space>
            </div>
            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table dataSource={logs} columns={columns} rowKey="id" loading={loading} pagination={false} locale={{ emptyText: <Empty description="暂无日志" /> }} />
            </div>
            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={(p) => fetchLogs(p)} showTotal={(t) => `共 ${t} 条日志`} />
                </div>
            )}
        </div>
    )
}
