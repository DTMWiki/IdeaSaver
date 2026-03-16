import { useEffect, useRef, useState } from 'react'
import { Card, Row, Col, Button, Space, Typography, Switch, Popconfirm, Empty, Spin, Modal, App, Pagination } from 'antd'
import {
    UploadOutlined,
    PlayCircleOutlined,
    DeleteOutlined,
    VideoCameraOutlined,
} from '@ant-design/icons'
import { useVideoStore } from '@/stores/videoStore'
import { getPlayURL } from '@/api/videos'
import { formatBytes, formatDate } from '@/utils/format'

const { Title, Text } = Typography

export default function VideoPage() {
    const { videos, total, loading, page, pageSize, fetchVideos, uploadVideo, toggleStatus, deleteVideo, batchDelete, setPage } = useVideoStore()
    const fileInputRef = useRef<HTMLInputElement>(null)
    const { message, modal } = App.useApp()
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
    const [playUrl, setPlayUrl] = useState<string | null>(null)

    useEffect(() => { fetchVideos(1) }, [])

    const handleUpload = () => fileInputRef.current?.click()

    const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return
        try {
            await uploadVideo(file)
            message.success('视频上传成功，等待转码...')
        } catch {
            message.error('上传失败')
        }
        e.target.value = ''
    }

    const handlePlay = async (id: string) => {
        try {
            const url = await getPlayURL(id)
            setPlayUrl(url)
        } catch {
            message.error('获取播放地址失败')
        }
    }

    const handleBatchDelete = () => {
        const ids = Array.from(selectedIds)
        if (ids.length === 0) return
        modal.confirm({
            title: '批量删除',
            content: `确定要删除 ${ids.length} 个视频吗？`,
            okText: '删除',
            okType: 'danger',
            cancelText: '取消',
            onOk: async () => {
                await batchDelete(ids)
                setSelectedIds(new Set())
                message.success('删除成功')
            },
        })
    }

    const toggleSelect = (id: string) => {
        const next = new Set(selectedIds)
        if (next.has(id)) next.delete(id)
        else next.add(id)
        setSelectedIds(next)
    }

    return (
        <div className="fade-in">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
                <Title level={4} style={{ margin: 0 }}>视频管理</Title>
                <Space>
                    {selectedIds.size > 0 && (
                        <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>
                            批量删除 ({selectedIds.size})
                        </Button>
                    )}
                    <Button type="primary" icon={<UploadOutlined />} onClick={handleUpload}>
                        上传视频
                    </Button>
                </Space>
            </div>

            {loading && videos.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
            ) : videos.length === 0 ? (
                <Empty description="暂无视频" />
            ) : (
                <>
                    <Row gutter={[16, 16]}>
                        {videos.map((video) => (
                            <Col xs={24} sm={12} lg={8} xl={6} key={video.id}>
                                <Card
                                    hoverable
                                    style={{ borderColor: selectedIds.has(video.id) ? '#1677ff' : undefined }}
                                    onClick={() => toggleSelect(video.id)}
                                    actions={[
                                        <Button type="text" icon={<PlayCircleOutlined />} onClick={(e) => { e.stopPropagation(); handlePlay(video.id) }} key="play">
                                            播放
                                        </Button>,
                                        <Popconfirm
                                            key="del"
                                            title="删除此视频？"
                                            onConfirm={(e) => { e?.stopPropagation(); deleteVideo(video.id) }}
                                            onCancel={(e) => e?.stopPropagation()}
                                        >
                                            <Button type="text" danger icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()}>
                                                删除
                                            </Button>
                                        </Popconfirm>,
                                    ]}
                                >
                                    <Card.Meta
                                        avatar={<VideoCameraOutlined style={{ fontSize: 24, color: '#eb2f96' }} />}
                                        title={
                                            <Text ellipsis={{ tooltip: video.title }} style={{ maxWidth: 180 }}>
                                                {video.title}
                                            </Text>
                                        }
                                        description={
                                            <Space direction="vertical" size={4}>
                                                <Text type="secondary" style={{ fontSize: 12 }}>{formatBytes(video.size)}</Text>
                                                <Text type="secondary" style={{ fontSize: 12 }}>{formatDate(video.created_at)}</Text>
                                                <div onClick={(e) => e.stopPropagation()}>
                                                    <Switch
                                                        size="small"
                                                        checked={video.status === 1}
                                                        onChange={() => toggleStatus(video.id, video.status)}
                                                        checkedChildren="启用"
                                                        unCheckedChildren="禁用"
                                                    />
                                                </div>
                                            </Space>
                                        }
                                    />
                                </Card>
                            </Col>
                        ))}
                    </Row>

                    {total > pageSize && (
                        <div style={{ textAlign: 'center', marginTop: 24 }}>
                            <Pagination current={page} total={total} pageSize={pageSize} onChange={setPage} showSizeChanger={false} />
                        </div>
                    )}
                </>
            )}

            <input ref={fileInputRef} type="file" accept="video/*" style={{ display: 'none' }} onChange={handleFileSelect} />

            {/* Play Modal */}
            <Modal
                title="视频播放"
                open={!!playUrl}
                onCancel={() => setPlayUrl(null)}
                footer={null}
                width={720}
                destroyOnClose
            >
                {playUrl && (
                    <video controls autoPlay style={{ width: '100%', maxHeight: '60vh' }}>
                        <source src={playUrl} />
                        您的浏览器不支持视频播放
                    </video>
                )}
            </Modal>
        </div>
    )
}
