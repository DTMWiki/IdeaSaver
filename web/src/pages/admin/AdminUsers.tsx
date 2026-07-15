import { useEffect, useState } from 'react'
import { Table, Button, Typography, Empty, App, Pagination, InputNumber, Modal, Tag, Space, Grid } from 'antd'
import type { Breakpoint } from 'antd'
import { EditOutlined, CalculatorOutlined } from '@ant-design/icons'
import type { User } from '@/types'
import { listUsers, updateUserQuota, recalcAllStorage } from '@/api/admin'
import { formatBytes, formatDate } from '@/utils/format'

const { Title, Text } = Typography
const TABLE_MD: Breakpoint[] = ['md']
const TABLE_LG: Breakpoint[] = ['lg']
const TABLE_XL: Breakpoint[] = ['xl']

export default function AdminUsers() {
    const [users, setUsers] = useState<User[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const pageSize = 50
    const { message } = App.useApp()
    const screens = Grid.useBreakpoint()
    const isMobile = !screens.md

    // Quota edit
    const [editUser, setEditUser] = useState<User | null>(null)
    const [quotaGB, setQuotaGB] = useState<number>(5)

    const fetchUsers = async (p: number) => {
        setLoading(true)
        try {
            const data = await listUsers((p - 1) * pageSize, pageSize)
            setUsers(data.users)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchUsers(1) }, [])

    const openQuotaEdit = (user: User) => {
        setEditUser(user)
        setQuotaGB(Math.round(user.storage_quota / (1024 * 1024 * 1024)))
    }

    const handleUpdateQuota = async () => {
        if (!editUser) return
        const quotaBytes = quotaGB * 1024 * 1024 * 1024
        try {
            await updateUserQuota(editUser.id, quotaBytes)
            message.success('配额已更新')
            setEditUser(null)
            fetchUsers(page)
        } catch {
            message.error('更新失败')
        }
    }

    const handleRecalcStorage = async () => {
        try {
            const n = await recalcAllStorage()
            message.success(`已重新计算 ${n} 个用户的存储占用（文件+视频）`)
            fetchUsers(page)
        } catch {
            message.error('配额对账失败')
        }
    }

    const columns = [
        { title: '用户名', dataIndex: 'username', key: 'username', width: 140 },
        { title: '显示名', dataIndex: 'display_name', key: 'display_name', ellipsis: true },
        { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true, responsive: TABLE_LG },
        {
            title: '角色',
            dataIndex: 'role',
            key: 'role',
            width: 80,
            responsive: TABLE_MD,
            render: (role: string) => role === 'admin' ? <Tag color="gold">管理员</Tag> : <Tag>用户</Tag>,
        },
        {
            title: '存储配额',
            key: 'quota',
            width: 180,
            responsive: TABLE_MD,
            render: (_: unknown, record: User) => {
                const percent = record.storage_quota > 0 ? Math.round((record.storage_used / record.storage_quota) * 100) : 0
                return (
                    <Text style={{ fontSize: 13 }}>
                        {formatBytes(record.storage_used)} / {formatBytes(record.storage_quota)} ({percent}%)
                    </Text>
                )
            },
        },
        {
            title: '注册时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            responsive: TABLE_XL,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: 72,
            render: (_: unknown, record: User) => (
                <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openQuotaEdit(record)}>
                    {!isMobile && '配额'}
                </Button>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
                <Title level={4} style={{ margin: 0 }}>用户管理</Title>
                <Button icon={<CalculatorOutlined />} onClick={() => { void handleRecalcStorage() }}>
                    重新计算存储占用
                </Button>
            </div>
            <div className="page-card">
                <Table
                    dataSource={users}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    size={isMobile ? 'small' : 'middle'}
                    scroll={isMobile ? { x: 760 } : { x: 980 }}
                    locale={{ emptyText: <Empty description="暂无用户" /> }}
                />
            </div>
            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={fetchUsers} showTotal={(t) => `共 ${t} 个用户`} />
                </div>
            )}

            {/* Quota Edit Modal */}
            <Modal
                title={`调整配额 — ${editUser?.username}`}
                open={!!editUser}
                onOk={handleUpdateQuota}
                onCancel={() => setEditUser(null)}
                okText="保存"
                cancelText="取消"
            >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Text>当前已使用: {editUser ? formatBytes(editUser.storage_used) : '-'}</Text>
                    <div>
                        <Text style={{ display: 'block', marginBottom: 4 }}>存储配额 (GB)</Text>
                        <InputNumber
                            min={1}
                            max={1024}
                            value={quotaGB}
                            onChange={(v) => setQuotaGB(v || 5)}
                            addonAfter="GB"
                            style={{ width: '100%' }}
                        />
                    </div>
                </Space>
            </Modal>
        </div>
    )
}
