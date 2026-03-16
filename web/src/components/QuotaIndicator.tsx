import { useEffect, useState } from 'react'
import { Progress, Tooltip, Space, Typography } from 'antd'
import { useAuthStore } from '@/stores/authStore'
import { getQuota } from '@/api/auth'
import { formatBytes } from '@/utils/format'

const { Text } = Typography

interface QuotaIndicatorProps {
    compact?: boolean
}

export default function QuotaIndicator({ compact = false }: QuotaIndicatorProps) {
    const { user } = useAuthStore()
    const [quota, setQuota] = useState(user?.storage_quota || 0)
    const [used, setUsed] = useState(user?.storage_used || 0)

    useEffect(() => {
        getQuota()
            .then(({ quota: q, used: u }) => {
                setQuota(q)
                setUsed(u)
            })
            .catch(() => { })
    }, [])

    const percent = quota > 0 ? Math.round((used / quota) * 100) : 0
    const strokeColor = percent > 90 ? '#ff4d4f' : percent > 70 ? '#faad14' : '#1677ff'

    if (compact) {
        return (
            <Tooltip title={`${formatBytes(used)} / ${formatBytes(quota)} (${percent}%)`}>
                <Progress
                    type="circle"
                    percent={percent}
                    size={28}
                    strokeColor={strokeColor}
                    format={() => `${percent}%`}
                    strokeWidth={8}
                />
            </Tooltip>
        )
    }

    return (
        <div>
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                    存储空间
                </Text>
                <Progress
                    percent={percent}
                    strokeColor={strokeColor}
                    size="small"
                    format={() => `${percent}%`}
                />
                <Text type="secondary" style={{ fontSize: 11 }}>
                    {formatBytes(used)} / {formatBytes(quota)}
                </Text>
            </Space>
        </div>
    )
}
