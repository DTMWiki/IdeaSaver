import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Input, Button, Typography, Space, Spin, Result, Image } from 'antd'
import { LockOutlined, FileOutlined, DownloadOutlined } from '@ant-design/icons'
import type { ShareFileView } from '@/types'
import { accessShare, buildShareDownloadURL } from '@/api/shares'
import { formatBytes } from '@/utils/format'
import './ShareAccess.css'

const { Title, Text, Paragraph } = Typography

export default function ShareAccess() {
    const { code } = useParams<{ code: string }>()
    const [file, setFile] = useState<ShareFileView | null>(null)
    const [loading, setLoading] = useState(false)
    const [needPassword, setNeedPassword] = useState(false)
    const [password, setPassword] = useState('')
    const [error, setError] = useState<string | null>(null)
    const [unlockedPassword, setUnlockedPassword] = useState<string | undefined>()

    const fetchShare = useCallback(async (pwd?: string) => {
        if (!code) return
        setLoading(true)
        setError(null)
        try {
            const data = await accessShare(code, pwd)
            setFile(data)
            setNeedPassword(false)
            setUnlockedPassword(pwd || undefined)
        } catch (err: unknown) {
            const response = (err as {
                response?: { data?: { error?: string; code?: string } }
            })?.response?.data
            const codeKey = response?.code || ''
            const msg = response?.error || '访问失败'
            if (codeKey === 'password_required' || codeKey === 'password_invalid' || msg.includes('密码')) {
                setNeedPassword(true)
                if (codeKey === 'password_invalid' || msg.includes('密码错误')) {
                    setError('密码错误')
                } else {
                    setError(null)
                }
            } else {
                setError(msg)
            }
        } finally {
            setLoading(false)
        }
    }, [code])

    useEffect(() => { void fetchShare() }, [fetchShare])

    const handleSubmitPassword = () => {
        fetchShare(password)
    }

    const downloadURL = useMemo(() => {
        if (!code || !file) return ''
        return buildShareDownloadURL(code, unlockedPassword)
    }, [code, file, unlockedPassword])

    if (loading && !needPassword) {
        return (
            <div className="share-access-container">
                <Spin size="large" />
            </div>
        )
    }

    if (error && !needPassword) {
        return (
            <div className="share-access-container">
                <Result status="error" title="访问失败" subTitle={error} />
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
                        <Text type="secondary">此分享受密码保护，下载也会再次校验密码</Text>
                        <Input.Password
                            placeholder="请输入访问密码"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            onPressEnter={handleSubmitPassword}
                            status={error ? 'error' : undefined}
                        />
                        {error && <Text type="danger">{error}</Text>}
                        <Button type="primary" block onClick={handleSubmitPassword} loading={loading}>
                            访问
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

                    {file.is_image && downloadURL && (
                        <Image src={downloadURL} alt={file.name} style={{ maxHeight: 400, objectFit: 'contain' }} />
                    )}

                    {downloadURL && (
                        <Button type="primary" icon={<DownloadOutlined />} href={downloadURL} target="_blank" block>
                            下载文件
                        </Button>
                    )}

                    <Paragraph type="secondary" style={{ fontSize: 12, textAlign: 'center', margin: 0 }}>
                        由 IdeaSaver 提供分享 · DTMWiki
                    </Paragraph>
                </Space>
            </Card>
        </div>
    )
}
