import { useState } from 'react'
import { Modal, Input, Select, Space, Typography, App, Alert } from 'antd'
import { createShare } from '@/api/files'
import { copyToClipboard } from '@/utils/format'

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

function expiresLabel(value: number) {
    return expiresOptions.find((item) => item.value === value)?.label || '自定义'
}

export default function CreateShareModal({ fileId, onClose }: CreateShareModalProps) {
    const [password, setPassword] = useState('')
    const [expiresIn, setExpiresIn] = useState(604800) // default 7 days
    const [loading, setLoading] = useState(false)
    const [shareUrl, setShareUrl] = useState<string | null>(null)
    const [createdPassword, setCreatedPassword] = useState('')
    const [createdExpires, setCreatedExpires] = useState(604800)
    const { message } = App.useApp()

    const handleCreate = async () => {
        if (!fileId) return
        setLoading(true)
        try {
            const share = await createShare(fileId, password || undefined, expiresIn || undefined)
            const url = `${window.location.origin}/share/${share.code}`
            setShareUrl(url)
            setCreatedPassword(password)
            setCreatedExpires(expiresIn)
            try {
                await copyToClipboard(url)
                message.success('分享已创建，链接已复制到剪贴板')
            } catch {
                message.success('分享链接已创建')
            }
        } catch {
            message.error('创建分享失败，请稍后重试')
        } finally {
            setLoading(false)
        }
    }

    const handleClose = () => {
        setPassword('')
        setExpiresIn(604800)
        setShareUrl(null)
        setCreatedPassword('')
        setCreatedExpires(604800)
        onClose()
    }

    return (
        <Modal
            title="创建分享链接"
            open={!!fileId}
            onOk={shareUrl ? handleClose : () => { void handleCreate() }}
            onCancel={handleClose}
            okText={shareUrl ? '完成' : '创建并复制链接'}
            cancelText="取消"
            confirmLoading={loading}
            destroyOnClose
        >
            {shareUrl ? (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Alert
                        type="success"
                        showIcon
                        message="分享已创建"
                        description={`有效期：${expiresLabel(createdExpires)}。对方打开链接即可下载；若设置了密码，访问与下载都会校验密码。`}
                    />
                    <div
                        style={{
                            padding: 12,
                            background: 'var(--color-bg-spotlight)',
                            borderRadius: 'var(--border-radius-sm)',
                        }}
                    >
                        <Text type="secondary" style={{ display: 'block', marginBottom: 6 }}>分享链接</Text>
                        <Paragraph copyable={{ text: shareUrl }} style={{ margin: 0, wordBreak: 'break-all' }}>
                            {shareUrl}
                        </Paragraph>
                    </div>
                    {createdPassword ? (
                        <div>
                            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>访问密码（请单独告知对方）</Text>
                            <Text code copyable>{createdPassword}</Text>
                        </div>
                    ) : (
                        <Text type="secondary">未设置密码，知道链接的人都可以下载</Text>
                    )}
                </Space>
            ) : (
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                    <Alert
                        type="info"
                        showIcon
                        message="适合发给同事或外链下载"
                        description="可设置密码和有效期。下载走受控接口，不会在创建时暴露永久直链。"
                    />
                    <div>
                        <Text style={{ display: 'block', marginBottom: 4 }}>访问密码（可选）</Text>
                        <Input.Password
                            placeholder="留空则无需密码"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            maxLength={64}
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
