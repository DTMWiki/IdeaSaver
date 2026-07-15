import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Input, Button, Typography, Space, Spin, Result, Image, App } from 'antd'
import { LockOutlined, FileOutlined, DownloadOutlined } from '@ant-design/icons'
import type { ShareFileView } from '@/types'
import { accessShare, buildShareDownloadURL } from '@/api/shares'
import { formatBytes } from '@/utils/format'
import './ShareAccess.css'

const { Title, Text, Paragraph } = Typography

export default function ShareAccess() {
    const { code } = useParams<{ code: string }>()
    const { message } = App.useApp()
    const [file, setFile] = useState<ShareFileView | null>(null)
    const [downloadToken, setDownloadToken] = useState<string | null>(null)
    const [loading, setLoading] = useState(false)
    const [needPassword, setNeedPassword] = useState(false)
    const [password, setPassword] = useState('')
    /** Kept in memory only to re-issue download tokens after expiry — never put in URLs. */
    const [unlockedPassword, setUnlockedPassword] = useState<string | undefined>()
    const [error, setError] = useState<string | null>(null)
    const [previewURL, setPreviewURL] = useState<string | null>(null)

    const fetchShare = useCallback(async (pwd?: string) => {
        if (!code) return
        setLoading(true)
        setError(null)
        try {
            const data = await accessShare(code, pwd)
            setFile(data.file)
            setDownloadToken(data.download_token)
            setUnlockedPassword(pwd || undefined)
            setNeedPassword(false)
        } catch (err: unknown) {
            const response = (err as {
                response?: { data?: { error?: string; code?: string } }
            })?.response?.data
            const codeKey = response?.code || ''
            const msg = response?.error || '访问失败'
            if (codeKey === 'password_required' || codeKey === 'password_invalid' || msg.includes('密码')) {
                setNeedPassword(true)
                setFile(null)
                setDownloadToken(null)
                if (codeKey === 'password_invalid' || msg.includes('密码错误')) {
                    setError('密码错误，请重试')
                } else {
                    setError(null)
                }
            } else {
                setError(msg)
                setNeedPassword(false)
            }
        } finally {
            setLoading(false)
        }
    }, [code])

    useEffect(() => { void fetchShare() }, [fetchShare])

    // Image preview via short-lived token (streamed; no password in URL).
    useEffect(() => {
        if (!code || !file?.is_image || !downloadToken) {
            setPreviewURL(null)
            return
        }
        setPreviewURL(buildShareDownloadURL(code, downloadToken, { inline: true }))
    }, [code, file, downloadToken])

    const handleSubmitPassword = () => {
        if (!password.trim()) {
            setError('请输入访问密码')
            return
        }
        void fetchShare(password)
    }

    // Native navigation download — browser streams to disk (works for large files).
    const handleDownload = async () => {
        if (!code || !file) {
            message.error('无法下载')
            return
        }
        let token = downloadToken
        if (!token) {
            try {
                const data = await accessShare(code, unlockedPassword)
                token = data.download_token
                setDownloadToken(token)
                setFile(data.file)
            } catch {
                message.error('下载凭证已失效，请重新打开分享')
                return
            }
        }
        const url = buildShareDownloadURL(code, token)
        const a = document.createElement('a')
        a.href = url
        a.download = file.name || 'download'
        a.rel = 'noopener'
        document.body.appendChild(a)
        a.click()
        a.remove()
    }

    if (loading && !needPassword) {
        return (
            <div className="share-access-container">
                <Spin size="large" tip="正在打开分享..." />
            </div>
        )
    }

    if (error && !needPassword) {
        return (
            <div className="share-access-container">
                <Result
                    status="error"
                    title="无法打开分享"
                    subTitle={error}
                    extra={
                        <Button type="primary" onClick={() => { void fetchShare() }}>
                            重试
                        </Button>
                    }
                />
            </div>
        )
    }

    if (needPassword && !file) {
        return (
            <div className="share-access-container">
                <Card className="share-card" style={{ maxWidth: 400, width: '100%' }}>
                    <Space direction="vertical" size={20} align="center" style={{ width: '100%' }}>
                        <LockOutlined style={{ fontSize: 48, color: '#1677ff' }} />
                        <Title level={4} style={{ margin: 0 }}>需要密码</Title>
                        <Text type="secondary">输入分享密码后即可查看并下载</Text>
                        <Input.Password
                            placeholder="请输入访问密码"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            onPressEnter={handleSubmitPassword}
                            status={error ? 'error' : undefined}
                            autoFocus
                        />
                        {error && <Text type="danger">{error}</Text>}
                        <Button type="primary" block onClick={handleSubmitPassword} loading={loading}>
                            打开分享
                        </Button>
                    </Space>
                </Card>
            </div>
        )
    }

    if (!file) return null

    return (
        <div className="share-access-container">
            <Card className="share-card" style={{ maxWidth: 600, width: '100%' }}>
                <Space direction="vertical" size={16} style={{ width: '100%' }}>
                    <Space>
                        <FileOutlined style={{ fontSize: 24, color: '#1677ff' }} />
                        <div>
                            <Title level={4} style={{ margin: 0 }}>{file.name}</Title>
                            <Text type="secondary">{formatBytes(file.size)}</Text>
                        </div>
                    </Space>

                    {file.is_image && previewURL && (
                        <Image src={previewURL} alt={file.name} style={{ maxHeight: 400, objectFit: 'contain' }} />
                    )}

                    <Button
                        type="primary"
                        icon={<DownloadOutlined />}
                        onClick={() => { void handleDownload() }}
                        block
                    >
                        下载文件
                    </Button>

                    <Paragraph type="secondary" style={{ fontSize: 12, textAlign: 'center', margin: 0 }}>
                        由 IdeaSaver 提供分享 · DTMWiki
                    </Paragraph>
                </Space>
            </Card>
        </div>
    )
}
