import { useEffect, useState } from 'react'
import { Timeline, Typography, Tag, Empty, Spin, Button, Space } from 'antd'
import {
    UploadOutlined,
    DeleteOutlined,
    EditOutlined,
    LoginOutlined,
    FileOutlined,
    VideoCameraOutlined,
    ShareAltOutlined,
    UserOutlined,
} from '@ant-design/icons'
import type { AuditLog } from '@/types'
import { getHistory } from '@/api/auth'
import { formatRelativeTime, formatDate } from '@/utils/format'

const { Title, Text } = Typography

const ACTION_CONFIG: Record<string, { icon: React.ReactNode; color: string; label: string }> = {
    login: { icon: <LoginOutlined />, color: '#1677ff', label: '登录' },
    upload: { icon: <UploadOutlined />, color: '#52c41a', label: '上传' },
    delete: { icon: <DeleteOutlined />, color: '#ff4d4f', label: '删除' },
    rename: { icon: <EditOutlined />, color: '#faad14', label: '重命名' },
    move: { icon: <FileOutlined />, color: '#722ed1', label: '移动' },
    copy: { icon: <FileOutlined />, color: '#13c2c2', label: '复制' },
    share: { icon: <ShareAltOutlined />, color: '#eb2f96', label: '分享' },
    restore: { icon: <FileOutlined />, color: '#52c41a', label: '恢复' },
    video_upload: { icon: <VideoCameraOutlined />, color: '#1677ff', label: '视频上传' },
    video_delete: { icon: <VideoCameraOutlined />, color: '#ff4d4f', label: '视频删除' },
}

const DEFAULT_CONFIG = { icon: <UserOutlined />, color: '#8c8c8c', label: '操作' }

export default function HistoryPage() {
    const [logs, setLogs] = useState<AuditLog[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [offset, setOffset] = useState(0)
    const limit = 30

    const fetchLogs = async (newOffset: number) => {
        setLoading(true)
        try {
            const data = await getHistory(newOffset, limit)
            if (newOffset === 0) {
                setLogs(data.logs)
            } else {
                setLogs((prev) => [...prev, ...data.logs])
            }
            setTotal(data.total)
            setOffset(newOffset)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchLogs(0) }, [])

    const handleLoadMore = () => {
        fetchLogs(offset + limit)
    }

    return (
        <div className="fade-in">
            <Title level={4} style={{ marginBottom: 16 }}>操作历史</Title>

            {logs.length === 0 && !loading ? (
                <Empty description="暂无操作记录" />
            ) : (
                <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)', padding: 24 }}>
                    <Timeline
                        items={logs.map((log) => {
                            const config = ACTION_CONFIG[log.action] || DEFAULT_CONFIG
                            return {
                                dot: config.icon,
                                color: config.color,
                                children: (
                                    <div style={{ paddingBottom: 4 }}>
                                        <Space size={8} wrap>
                                            <Tag color={config.color}>{config.label}</Tag>
                                            {log.resource && (
                                                <Text type="secondary" style={{ fontSize: 13 }}>
                                                    {log.resource}
                                                </Text>
                                            )}
                                        </Space>
                                        {log.details && typeof log.details === 'object' && (
                                            <Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 2 }}>
                                                {JSON.stringify(log.details)}
                                            </Text>
                                        )}
                                        <Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 2 }}>
                                            {formatRelativeTime(log.created_at)} · {formatDate(log.created_at)}
                                        </Text>
                                        {log.ip_address && (
                                            <Text type="secondary" style={{ display: 'block', fontSize: 11 }}>
                                                IP: {log.ip_address}
                                            </Text>
                                        )}
                                    </div>
                                ),
                            }
                        })}
                    />

                    {loading && (
                        <div style={{ textAlign: 'center', padding: 16 }}><Spin /></div>
                    )}

                    {!loading && logs.length < total && (
                        <div style={{ textAlign: 'center' }}>
                            <Button onClick={handleLoadMore}>加载更多</Button>
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}
