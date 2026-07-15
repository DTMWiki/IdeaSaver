import { useState } from 'react'
import { Modal, Input, Select, Space, Typography, App } from 'antd'
import { createShare } from '@/api/files'

const { Text, Paragraph } = Typography

interface CreateShareModalProps {
    fileId: string | null
    onClose: () => void
}

// values are seconds; backend CreateShare uses expires_in as seconds
const expiresOptions = [
    { label: '1 小时', value: 3600 },
    { label: '1 天', value: 86400 },
    { label: '7 天', value: 604800 },
    { label: '30 天', value: 2592000 },
    { label: '永久', value: 0 },
]

export default function CreateShareModal({ fileId, onClose }: CreateShareModalProps) {
    const [password, setPassword] = useState('')
    const [expiresIn, setExpiresIn] = useState(604800) // default 7 days
    const [loading, setLoading] = useState(false)
    const [shareUrl, setShareUrl] = useState<string | null>(null)
    const { message } = App.useApp()

    const handleCreate = async () => {
        if (!fileId) return
        setLoading(true)
        try {
            const share = await createShare(fileId, password || undefined, expiresIn || undefined)
            const url = `${window.location.origin}/share/${share.code}`
            setShareUrl(url)
            message.success('分享链接已创建')
        } catch {
            message.error('创建分享失败')
        } finally {
            setLoading(false)
        }
    }

    const handleClose = () => {
        setPassword('')
        setExpiresIn(604800)
        setShareUrl(null)
        onClose()
    }

    return (
        <Modal
            title="创建分享链接"
            open={!!fileId}
            onOk={shareUrl ? handleClose : handleCreate}
            onCancel={handleClose}
            okText={shareUrl ? '完成' : '创建'}
            cancelText="取消"
            confirmLoading={loading}
            destroyOnClose
        >
            {shareUrl ? (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Text type="success">分享链接已创建！</Text>
                    <div
                        style={{
                            padding: 12,
                            background: 'var(--color-bg-spotlight)',
                            borderRadius: 'var(--border-radius-sm)',
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                        }}
                    >
                        <Paragraph copyable={{ text: shareUrl }} style={{ margin: 0, flex: 1, wordBreak: 'break-all' }}>
                            {shareUrl}
                        </Paragraph>
                    </div>
                    {password && (
                        <Text type="secondary">
                            密码: <Text code copyable>{password}</Text>
                        </Text>
                    )}
                </Space>
            ) : (
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                    <div>
                        <Text style={{ display: 'block', marginBottom: 4 }}>访问密码（可选）</Text>
                        <Input.Password
                            placeholder="留空则无需密码"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                        />
                    </div>
                    <div>
                        <Text style={{ display: 'block', marginBottom: 4 }}>有效期</Text>
                        <Select
                            value={expiresIn}
                            onChange={setExpiresIn}
                            options={expiresOptions}
                            style={{ width: '100%' }}
                        />
                    </div>
                </Space>
            )}
        </Modal>
    )
}
