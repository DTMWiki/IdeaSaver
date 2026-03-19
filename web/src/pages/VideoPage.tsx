import { useEffect, useRef, useState } from 'react'
import { Card, Row, Col, Button, Space, Typography, Switch, Popconfirm, Empty, Spin, Modal, App, Pagination, Alert, Progress, Tag } from 'antd'
import {
    UploadOutlined,
    PlayCircleOutlined,
    DeleteOutlined,
} from '@ant-design/icons'
import { useVideoStore } from '@/stores/videoStore'
import { getPlayInfo, type VideoPlayInfo, type VideoUploadProgress } from '@/api/videos'
import { formatBytes, formatDate } from '@/utils/format'
import VideoThumbnail from '@/components/VideoThumbnail'
import type { Video } from '@/types'

const { Title, Text } = Typography

const DOGE_PLAYER_SCRIPT = '/vendor/dogeplayer-loader.js'
let dogePlayerLoader: Promise<void> | null = null

type DogePlayerOptions = {
    container: HTMLDivElement
    vcode: string
    userId: number
    autoPlay?: boolean
}

type DogePlayerInstance = {
    destroy?: () => void
}

type DogePlayerConstructor = new (options: DogePlayerOptions) => DogePlayerInstance

type DogePlayerWindow = Window & {
    DogePlayer?: DogePlayerConstructor
    DogeCloudPlayer?: DogePlayerConstructor
}

function loadDogePlayerScript() {
    if (dogePlayerLoader) return dogePlayerLoader
    dogePlayerLoader = new Promise((resolve, reject) => {
        const existing = document.querySelector(`script[src="${DOGE_PLAYER_SCRIPT}"]`) as HTMLScriptElement | null
        if (existing) {
            const playerWindow = window as DogePlayerWindow
            if (playerWindow.DogePlayer || playerWindow.DogeCloudPlayer) {
                resolve()
                return
            }
            existing.addEventListener('load', () => resolve())
            existing.addEventListener('error', () => reject(new Error('加载 DogePlayer 脚本失败')))
            return
        }

        const script = document.createElement('script')
        script.src = DOGE_PLAYER_SCRIPT
        script.async = true
        script.onload = () => resolve()
        script.onerror = () => reject(new Error('加载 DogePlayer 脚本失败'))
        document.head.appendChild(script)
    })
    return dogePlayerLoader
}

export default function VideoPage() {
    const { videos, total, loading, page, pageSize, fetchVideos, uploadVideo, toggleStatus, deleteVideo, batchDelete, setPage } = useVideoStore()
    const fileInputRef = useRef<HTMLInputElement>(null)
    const playerContainerRef = useRef<HTMLDivElement>(null)
    const { message, modal } = App.useApp()
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
    const [playInfo, setPlayInfo] = useState<VideoPlayInfo | null>(null)
    const [useNativePlayer, setUseNativePlayer] = useState(true)
    const [sdkError, setSdkError] = useState<string | null>(null)
    const [uploadingFileName, setUploadingFileName] = useState('')
    const [uploadProgress, setUploadProgress] = useState<VideoUploadProgress | null>(null)

    useEffect(() => { fetchVideos(1) }, [fetchVideos])

    useEffect(() => {
        if (!playInfo || !playInfo.ready) return

        const vcode = playInfo.vcode?.trim()
        const playerUserID = firstNonEmptyString(
            playInfo.player_user_id,
            playerUserIDFromPlayURL(playInfo.play_url),
        )
        const playerUserIDNum = Number(playerUserID)
        if (!vcode || !playerUserID || Number.isNaN(playerUserIDNum) || playerUserIDNum <= 0) {
            setUseNativePlayer(true)
            return
        }

        let disposed = false
        let playerInstance: DogePlayerInstance | null = null
        const playerContainer = playerContainerRef.current
        setSdkError(null)

        loadDogePlayerScript()
            .then(() => {
                if (disposed) return
                const playerWindow = window as DogePlayerWindow
                const DogePlayer = playerWindow.DogePlayer || playerWindow.DogeCloudPlayer
                if (!DogePlayer || !playerContainer) {
                    setUseNativePlayer(true)
                    setSdkError('DogePlayer SDK 未注入，已切换为原生播放器')
                    return
                }

                try {
                    playerContainer.innerHTML = ''
                    playerInstance = new DogePlayer({
                        container: playerContainer,
                        vcode,
                        userId: playerUserIDNum,
                        autoPlay: true,
                    })
                    setUseNativePlayer(false)
                } catch {
                    setUseNativePlayer(true)
                    setSdkError('DogePlayer 初始化失败，已切换为原生播放器')
                }
            })
            .catch(() => {
                if (!disposed) {
                    setUseNativePlayer(true)
                    setSdkError('加载 DogePlayer 失败，已切换为原生播放器')
                }
            })

        return () => {
            disposed = true
            if (playerInstance && typeof playerInstance.destroy === 'function') {
                playerInstance.destroy()
            }
            if (playerContainer) {
                playerContainer.innerHTML = ''
            }
        }
    }, [playInfo])

    const handleUpload = () => fileInputRef.current?.click()

    const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return

        setUploadingFileName(file.name)
        setUploadProgress({ percent: 1, phase: 'uploading' })
        try {
            await uploadVideo(file, undefined, {
                onProgress: (progress) => {
                    setUploadProgress(progress)
                },
            })
            message.success('视频已上传，等待多吉云回调转码完成后即可播放')
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '视频上传失败')
        } finally {
            setUploadProgress(null)
            setUploadingFileName('')
        }
        e.target.value = ''
    }

    const handlePlay = async (id: string) => {
        try {
            const info = await getPlayInfo(id)
            if (!info.ready) {
                message.info(info.message || '视频仍在转码处理中，请稍后重试')
                return
            }
            if (!info.play_url && !(info.vcode && info.player_user_id)) {
                message.error('视频播放信息不完整，请稍后重试')
                return
            }
            setPlayInfo(info)
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            message.error(maybeMessage || '获取播放地址失败')
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

            {uploadProgress && (
                <Card style={{ marginBottom: 16, borderColor: 'var(--color-border-secondary)' }}>
                    <Space direction="vertical" size={6} style={{ width: '100%' }}>
                        <Text strong>{uploadingFileName}</Text>
                        <Progress percent={uploadProgress.percent} status="active" />
                        <Text type="secondary">
                            {uploadProgress.phase === 'uploading'
                                ? '正在上传到业务服务器'
                                : '文件已送达，正在写入多吉云并等待转码任务建立'}
                        </Text>
                    </Space>
                </Card>
            )}

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
                                    cover={(
                                        <VideoThumbnail
                                            src={video.thumbnail_small_url || video.thumbnail_url}
                                            alt={video.title}
                                            height={160}
                                            borderRadius={0}
                                            iconSize={30}
                                        />
                                    )}
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
                                        title={(
                                            <Space direction="vertical" size={4} style={{ width: '100%' }}>
                                                <Text ellipsis={{ tooltip: video.title }} style={{ maxWidth: 180 }}>
                                                    {video.title}
                                                </Text>
                                                {renderTranscodeTag(video)}
                                            </Space>
                                        )}
                                        description={(
                                            <Space direction="vertical" size={4}>
                                                <Text type="secondary" style={{ fontSize: 12 }}>{formatBytes(video.size)}</Text>
                                                <Text type="secondary" style={{ fontSize: 12 }}>{formatDate(video.created_at)}</Text>
                                                {video.transcode_message && (
                                                    <Text type="secondary" style={{ fontSize: 12 }}>
                                                        {video.transcode_message}
                                                    </Text>
                                                )}
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
                                        )}
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

            <Modal
                title="视频播放"
                open={!!playInfo}
                onCancel={() => {
                    setPlayInfo(null)
                    setUseNativePlayer(true)
                    setSdkError(null)
                }}
                footer={null}
                width={820}
                destroyOnClose
            >
                {playInfo && (
                    <>
                        {playInfo.message && !playInfo.ready && (
                            <Alert
                                type="info"
                                showIcon
                                style={{ marginBottom: 12 }}
                                message={playInfo.message}
                            />
                        )}
                        {sdkError && (
                            <Alert
                                type="warning"
                                showIcon
                                style={{ marginBottom: 12 }}
                                message={sdkError}
                            />
                        )}
                        {!useNativePlayer && playInfo.vcode && playInfo.player_user_id ? (
                            <div
                                ref={playerContainerRef}
                                style={{ width: '100%', minHeight: 420, background: '#000' }}
                            />
                        ) : (
                            <video controls autoPlay style={{ width: '100%', maxHeight: '70vh' }}>
                                <source src={playInfo.play_url} />
                                您的浏览器不支持视频播放
                            </video>
                        )}
                    </>
                )}
            </Modal>
        </div>
    )
}

function renderTranscodeTag(video: Video) {
    switch (video.transcode_status) {
    case 'ready':
        return <Tag color="green">可播放</Tag>
    case 'failed':
        return <Tag color="red">转码失败</Tag>
    case 'blocked':
        return <Tag color="volcano">已屏蔽</Tag>
    case 'processing':
        return <Tag color="blue">转码中</Tag>
    default:
        return <Tag>排队中</Tag>
    }
}

function firstNonEmptyString(...values: Array<string | undefined>) {
    for (const value of values) {
        const v = value?.trim()
        if (v) {
            return v
        }
    }
    return ''
}

function playerUserIDFromPlayURL(raw?: string) {
    const value = raw?.trim()
    if (!value) return ''
    try {
        const u = new URL(value)
        return firstNonEmptyString(
            u.searchParams.get('userId') ?? '',
            u.searchParams.get('userid') ?? '',
            u.searchParams.get('uid') ?? '',
        )
    } catch {
        return ''
    }
}
