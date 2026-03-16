import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Input, Button, Typography, Space, Spin, Result, Image } from 'antd'
import { LockOutlined, FileOutlined, DownloadOutlined } from '@ant-design/icons'
import type { FileItem } from '@/types'
import { accessShare } from '@/api/shares'
import { formatBytes, isImage } from '@/utils/format'
import './ShareAccess.css'

const { Title, Text, Paragraph } = Typography

export default function ShareAccess() {
    const { code } = useParams<{ code: string }>()
    const [file, setFile] = useState<FileItem | null>(null)
    const [loading, setLoading] = useState(false)
    const [needPassword, setNeedPassword] = useState(false)
    const [password, setPassword] = useState('')
    const [error, setError] = useState<string | null>(null)

    const fetchShare = async (pwd?: string) => {
        if (!code) return
        setLoading(true)
        setError(null)
        try {
            const data = await accessShare(code, pwd)
            setFile(data)
            setNeedPassword(false)
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '访问失败'
            if (msg.includes('密码') || msg.includes('password')) {
                setNeedPassword(true)
            } else {
                setError(msg)
            }
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchShare() }, [code])

    const handleSubmitPassword = () => {
        fetchShare(password)
    }

    if (loading && !needPassword) {
        return (
            <div className="share-access-container">
                <Spin size="large" />
            </div>
        )
    }

    if (error) {
        return (
            <div className="share-access-container">
                <Result status="error" title="访问失败" subTitle={error} />
            </div>
        )
    }

    if (needPassword) {
        return (
            <div className="share-access-container">
                <Card className="share-card" style={{ maxWidth: 400, width: '100%' }}>
                    <Space direction="vertical" size={20} align="center" style={{ width: '100%' }}>
                        <LockOutlined style={{ fontSize: 48, color: '#1677ff' }} />
                        <Title level={4} style={{ margin: 0 }}>需要密码</Title>
                        <Text type="secondary">此分享链接受密码保护</Text>
                        <Input.Password
                            placeholder="请输入访问密码"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            onPressEnter={handleSubmitPassword}
                        />
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

                    {isImage(file.mime_type) && file.public_url && (
                        <Image src={file.public_url} alt={file.name} style={{ maxHeight: 400, objectFit: 'contain' }} />
                    )}

                    {file.public_url && (
                        <Button type="primary" icon={<DownloadOutlined />} href={file.public_url} target="_blank" block>
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
