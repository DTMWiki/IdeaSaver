import { Button, Typography, Space, Card, Row, Col } from 'antd'
import {
    CloudUploadOutlined,
    FileProtectOutlined,
    VideoCameraOutlined,
    ShareAltOutlined,
    ThunderboltOutlined,
    SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '@/stores/authStore'
import { useNavigate } from 'react-router-dom'
import './Home.css'

const { Title, Paragraph, Text } = Typography

const features = [
    {
        icon: <CloudUploadOutlined style={{ fontSize: 32, color: '#1677ff' }} />,
        title: '分片上传',
        desc: '大文件分片上传，支持暂停/续传，多任务并发',
    },
    {
        icon: <FileProtectOutlined style={{ fontSize: 32, color: '#722ed1' }} />,
        title: '文件管理',
        desc: '文件夹管理、批量操作、回收站、直链分发',
    },
    {
        icon: <VideoCameraOutlined style={{ fontSize: 32, color: '#eb2f96' }} />,
        title: '视频托管',
        desc: '视频上传自动转码、多清晰度播放',
    },
    {
        icon: <ShareAltOutlined style={{ fontSize: 32, color: '#52c41a' }} />,
        title: '安全分享',
        desc: '带密码和有效期的分享链接',
    },
    {
        icon: <ThunderboltOutlined style={{ fontSize: 32, color: '#faad14' }} />,
        title: '实时通知',
        desc: 'SSE 实时推送上传完成、转码进度',
    },
    {
        icon: <SafetyCertificateOutlined style={{ fontSize: 32, color: '#13c2c2' }} />,
        title: '安全可靠',
        desc: 'Authelia OAuth2 统一认证，审计日志全记录',
    },
]

export default function Home() {
    const { isLoggedIn, login } = useAuthStore()
    const navigate = useNavigate()

    return (
        <div className="home-container">
            {/* Hero Section */}
            <section className="hero-section">
                <div className="hero-bg" />
                <div className="hero-content">
                    <Space direction="vertical" size={16} align="center">
                        <div className="hero-icon">💡</div>
                        <Title level={1} className="hero-title">
                            IdeaSaver
                        </Title>
                        <Paragraph className="hero-subtitle">
                            DTMWiki 文件上传分发平台
                        </Paragraph>
                        <Text type="secondary" className="hero-desc">
                            高性能分片上传 · 视频云托管 · 直链分发 · 安全分享
                        </Text>
                        <Space size={16} style={{ marginTop: 16 }}>
                            {isLoggedIn ? (
                                <Button type="primary" size="large" onClick={() => navigate('/dashboard')} icon={<CloudUploadOutlined />}>
                                    进入控制台
                                </Button>
                            ) : (
                                <Button type="primary" size="large" onClick={login} icon={<SafetyCertificateOutlined />}>
                                    使用 Authelia 登录
                                </Button>
                            )}
                        </Space>
                    </Space>
                </div>
            </section>

            {/* Features Section */}
            <section className="features-section">
                <Title level={2} style={{ textAlign: 'center', marginBottom: 48 }}>
                    核心功能
                </Title>
                <Row gutter={[24, 24]} justify="center">
                    {features.map((feat) => (
                        <Col xs={24} sm={12} lg={8} key={feat.title}>
                            <Card className="feature-card" hoverable>
                                <Space direction="vertical" size={12} align="center" style={{ width: '100%' }}>
                                    {feat.icon}
                                    <Title level={4} style={{ margin: 0 }}>
                                        {feat.title}
                                    </Title>
                                    <Text type="secondary">{feat.desc}</Text>
                                </Space>
                            </Card>
                        </Col>
                    ))}
                </Row>
            </section>

            {/* Footer */}
            <footer className="home-footer">
                <Text type="secondary">
                    © {new Date().getFullYear()} DTMWiki · IdeaSaver · MIT License
                </Text>
            </footer>
        </div>
    )
}
