import { useCallback, useEffect, useRef, useState } from 'react'
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
    const [tokenExpiresAt, setTokenExpiresAt] = useState<number | null>(null)
    const [loading, setLoading] = useState(false)
    const [needPassword, setNeedPassword] = useState(false)
    const [password, setPassword] = useState('')
    /** Kept in memory only to re-issue download tokens after expiry — never put in URLs. */
    const [unlockedPassword, setUnlockedPassword] = useState<string | undefined>()
    const [error, setError] = useState<string | null>(null)
    const [previewURL, setPreviewURL] = useState<string | null>(null)
    const refreshTimerRef = useRef<number | null>(null)

    const applyAccessResult = useCallback((
        data: { file: ShareFileView; download_token: string; token_expires_in: number },
        pwd?: string,
    ) => {
        setFile(data.file)
        setDownloadToken(data.download_token)
        const ttl = Math.max(30, Number(data.token_expires_in) || 900)
        // Refresh a bit before expiry so download/preview keep working on long pages.
        setTokenExpiresAt(Date.now() + ttl * 1000)
        setUnlockedPassword(pwd)
        setNeedPassword(false)
    }, [])

    const fetchShare = useCallback(async (pwd?: string, opts?: { silent?: boolean }) => {
        if (!code) return
        if (!opts?.silent) {
            setLoading(true)
            setError(null)
        }
        try {
            const data = await accessShare(code, pwd)
            applyAccessResult(data, pwd)
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
                setTokenExpiresAt(null)
                if (codeKey === 'password_invalid' || msg.includes('密码错误')) {
                    setError('密码错误，请重试')
                } else if (!opts?.silent) {
                    setError(null)
                }
            } else if (!opts?.silent) {
                setError(msg)
                setNeedPassword(false)
            }
        } finally {
            if (!opts?.silent) setLoading(false)
        }
    }, [applyAccessResult, code])

    useEffect(() => {
        // Defer so React 19 set-state-in-effect lint stays happy for mount fetches.
        const t = window.setTimeout(() => { void fetchShare() }, 0)
        return () => window.clearTimeout(t)
    }, [fetchShare])

    // Proactively refresh short-lived download token before expiry.
    useEffect(() => {
        if (refreshTimerRef.current) {
            window.clearTimeout(refreshTimerRef.current)
            refreshTimerRef.current = null
        }
        if (!tokenExpiresAt || !file) return

        const ms = Math.max(5_000, tokenExpiresAt - Date.now() - 60_000)
        refreshTimerRef.current = window.setTimeout(() => {
            void fetchShare(unlockedPassword, { silent: true })
        }, ms)

        return () => {
            if (refreshTimerRef.current) {
                window.clearTimeout(refreshTimerRef.current)
                refreshTimerRef.current = null
            }
        }
    }, [tokenExpiresAt, file, unlockedPassword, fetchShare])

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
        const nearExpiry = tokenExpiresAt != null && tokenExpiresAt - Date.now() < 30_000
        if (!token || nearExpiry) {
            try {
                const data = await accessShare(code, unlockedPassword)
                applyAccessResult(data, unlockedPassword)
                token = data.download_token
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
