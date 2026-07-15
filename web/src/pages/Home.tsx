import { useEffect } from 'react'
import { Button, Typography, Space, Card, Row, Col } from 'antd'
import {
    CloudUploadOutlined,
    FileProtectOutlined,
    VideoCameraOutlined,
    ShareAltOutlined,
    ThunderboltOutlined,
    SafetyCertificateOutlined,
} from '@ant-design/icons'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faLightbulb } from '@fortawesome/free-solid-svg-icons'
import { useAuthStore } from '@/stores/authStore'
import { useNavigate } from 'react-router-dom'
import './Home.css'

const { Title, Paragraph, Text } = Typography

const features = [
    {
        icon: <CloudUploadOutlined style={{ fontSize: 32, color: '#1677ff' }} />,
        title: '大文件上传',
        desc: '支持暂停与续传，适合文档、图片和安装包',
    },
    {
        icon: <FileProtectOutlined style={{ fontSize: 32, color: '#722ed1' }} />,
        title: '文件整理',
        desc: '文件夹、批量操作、回收站与直链复制',
    },
    {
        icon: <VideoCameraOutlined style={{ fontSize: 32, color: '#eb2f96' }} />,
        title: '视频托管',
        desc: '上传后自动转码，可在线播放与嵌入',
    },
    {
        icon: <ShareAltOutlined style={{ fontSize: 32, color: '#52c41a' }} />,
        title: '受控分享',
        desc: '可为分享页设置密码与有效期，下载经服务端校验',
    },
    {
        icon: <ThunderboltOutlined style={{ fontSize: 32, color: '#faad14' }} />,
        title: '进度提醒',
        desc: '上传完成、转码结果会及时通知',
    },
    {
        icon: <SafetyCertificateOutlined style={{ fontSize: 32, color: '#13c2c2' }} />,
        title: '统一登录',
        desc: '使用组织账号登录，操作可追溯',
    },
]

export default function Home() {
    const { isLoggedIn, login, bootstrapped, bootstrap } = useAuthStore()
    const navigate = useNavigate()

    useEffect(() => {
        if (!bootstrapped) {
            void bootstrap()
        }
    }, [bootstrapped, bootstrap])

    return (
        <div className="home-container">
            <section className="hero-section">
                <div className="hero-bg" />
                <div className="hero-content">
                    <Space direction="vertical" size={16} align="center">
                        <div className="hero-mark" aria-hidden="true">
                            <FontAwesomeIcon icon={faLightbulb} />
                        </div>
                        <Title level={1} className="hero-title">
                            IdeaSaver
                        </Title>
                        <Paragraph className="hero-subtitle">
                            DTMWiki 文件与视频分发
                        </Paragraph>
                        <Text type="secondary" className="hero-desc">
                            上传、整理、分享与在线播放，集中在一处完成
                        </Text>
                        <Space size={16} style={{ marginTop: 16 }}>
                            {isLoggedIn ? (
                                <Button type="primary" size="large" onClick={() => navigate('/dashboard')} icon={<CloudUploadOutlined />}>
                                    进入控制台
                                </Button>
                            ) : (
                                <Button type="primary" size="large" onClick={login} icon={<SafetyCertificateOutlined />}>
                                    登录
                                </Button>
                            )}
                        </Space>
                    </Space>
                </div>
            </section>

            <section className="features-section">
                <Title level={2} style={{ textAlign: 'center', marginBottom: 48 }}>
                    能做什么
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

            <footer className="home-footer">
                <Text type="secondary">
                    © {new Date().getFullYear()} DTMWiki · IdeaSaver · MIT License
                </Text>
            </footer>
        </div>
    )
}
