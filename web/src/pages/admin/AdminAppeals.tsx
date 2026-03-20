import { useEffect, useState } from 'react'
import { Table, Typography, Empty, Select, Pagination, Tag, Space, Button, App, Modal, Input, Tooltip, Grid } from 'antd'
import type { Breakpoint } from 'antd'
import type { FileAppeal } from '@/types'
import { listAppeals, reviewAppeal } from '@/api/admin'
import { formatDate } from '@/utils/format'

const { Title, Text } = Typography
const TABLE_MD: Breakpoint[] = ['md']
const TABLE_LG: Breakpoint[] = ['lg']
const TABLE_XL: Breakpoint[] = ['xl']

const STATUS_OPTIONS = [
    { label: '全部', value: '' },
    { label: '待审核', value: 'pending' },
    { label: '已通过', value: 'approved' },
    { label: '已删除', value: 'deleted' },
    { label: '已驳回', value: 'rejected' },
]

const STATUS_COLOR: Record<string, string> = {
    pending: 'gold',
    approved: 'green',
    deleted: 'red',
    rejected: 'default',
}

const STATUS_LABEL: Record<string, string> = {
    pending: '待审核',
    approved: '已通过',
    deleted: '已删除',
    rejected: '已驳回',
}

type ReviewDecision = 'approve' | 'delete'

export default function AdminAppeals() {
    const [appeals, setAppeals] = useState<FileAppeal[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const [statusFilter, setStatusFilter] = useState('')
    const [reviewTarget, setReviewTarget] = useState<FileAppeal | null>(null)
    const [reviewDecision, setReviewDecision] = useState<ReviewDecision>('approve')
    const [reviewComment, setReviewComment] = useState('')
    const [reviewLoading, setReviewLoading] = useState(false)
    const pageSize = 50
    const { message } = App.useApp()
    const screens = Grid.useBreakpoint()
    const isMobile = !screens.md

    const fetchData = async (p: number, status?: string) => {
        setLoading(true)
        try {
            const s = status !== undefined ? status : statusFilter
            const data = await listAppeals(s || undefined, (p - 1) * pageSize, pageSize)
            setAppeals(data.appeals)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchData(1)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const handleReview = async () => {
        if (!reviewTarget) return
        setReviewLoading(true)
        try {
            await reviewAppeal(reviewTarget.id, reviewDecision, reviewComment.trim() || undefined)
            message.success(reviewDecision === 'approve' ? '已通过申诉并恢复访问' : '已彻底删除该资源')
            setReviewTarget(null)
            setReviewComment('')
            fetchData(page)
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '审核失败')
        } finally {
            setReviewLoading(false)
        }
    }

    const columns = [
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (status: string) => <Tag color={STATUS_COLOR[status] || 'default'}>{STATUS_LABEL[status] || status}</Tag>,
        },
        {
            title: '文件',
            dataIndex: 'file_name',
            key: 'file_name',
            ellipsis: true,
            render: (name: string) => name || '-',
        },
        {
            title: '申诉人',
            dataIndex: 'username',
            key: 'username',
            width: 120,
            responsive: TABLE_MD,
            render: (username: string) => username || '-',
        },
        {
            title: '申诉理由',
            dataIndex: 'reason',
            key: 'reason',
            ellipsis: true,
            render: (reason: string) => <Tooltip title={reason}>{reason}</Tooltip>,
        },
        {
            title: '处理意见',
            dataIndex: 'admin_comment',
            key: 'admin_comment',
            ellipsis: true,
            responsive: TABLE_LG,
            render: (comment: string) => comment ? <Tooltip title={comment}>{comment}</Tooltip> : '-',
        },
        {
            title: '提交时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 170,
            responsive: TABLE_XL,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: isMobile ? 124 : 180,
            render: (_: unknown, record: FileAppeal) => (
                record.status === 'pending' ? (
                    <Space size={4}>
                        <Button
                            type="link"
                            size="small"
                            onClick={() => {
                                setReviewTarget(record)
                            setReviewDecision('approve')
                            setReviewComment('')
                        }}
                    >
                            {!isMobile && '通过'}
                        </Button>
                        <Button
                            type="link"
                            size="small"
                            danger
                            onClick={() => {
                                setReviewTarget(record)
                            setReviewDecision('delete')
                            setReviewComment('')
                        }}
                    >
                            {!isMobile && '彻底删除'}
                        </Button>
                    </Space>
                ) : (
                    <Text type="secondary">已处理</Text>
                )
            ),
        },
    ]

    return (
        <div className="fade-in">
            <div className="page-header-bar">
                <Title level={4} style={{ margin: 0 }}>申诉工单</Title>
                <Space className="page-header-actions" wrap>
                    <Text type="secondary">状态筛选:</Text>
                    <Select
                        value={statusFilter}
                        onChange={(v) => {
                            setStatusFilter(v)
                            fetchData(1, v)
                        }}
                        options={STATUS_OPTIONS}
                        size="small"
                        style={{ width: 140 }}
                    />
                </Space>
            </div>

            <div className="page-card">
                <Table
                    dataSource={appeals}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    size={isMobile ? 'small' : 'middle'}
                    scroll={isMobile ? { x: 820 } : { x: 1040 }}
                    locale={{ emptyText: <Empty description="暂无申诉工单" /> }}
                />
            </div>

            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={(p) => fetchData(p)} showTotal={(t) => `共 ${t} 条工单`} />
                </div>
            )}

            <Modal
                title={reviewDecision === 'approve' ? '通过申诉并恢复访问' : '彻底删除资源'}
                open={!!reviewTarget}
                onOk={handleReview}
                onCancel={() => {
                    setReviewTarget(null)
                    setReviewComment('')
                }}
                confirmLoading={reviewLoading}
                okText={reviewDecision === 'approve' ? '确认通过' : '确认删除'}
                cancelText="取消"
                okButtonProps={reviewDecision === 'delete' ? { danger: true } : undefined}
                destroyOnClose
            >
                <Space direction="vertical" style={{ width: '100%' }} size={12}>
                    <Text>
                        目标文件: <Text strong>{reviewTarget?.file_name || '-'}</Text>
                    </Text>
                    <Text type="secondary">
                        申诉理由: {reviewTarget?.reason || '-'}
                    </Text>
                    <Input.TextArea
                        value={reviewComment}
                        onChange={(e) => setReviewComment(e.target.value)}
                        placeholder="可选：填写处理意见（会写入日志）"
                        rows={4}
                        maxLength={1000}
                        showCount
                    />
                </Space>
            </Modal>
        </div>
    )
}
