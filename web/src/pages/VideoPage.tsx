import { useEffect, useMemo, useRef, useState } from 'react'
import { Card, Row, Col, Button, Space, Typography, Switch, Popconfirm, Empty, Spin, Modal, App, Pagination, Alert, Tag } from 'antd'
import {
    UploadOutlined,
    PlayCircleOutlined,
    DeleteOutlined,
} from '@ant-design/icons'
import { useVideoStore } from '@/stores/videoStore'
import { getPlayInfo, type VideoPlayInfo } from '@/api/videos'
import { useUploadStore } from '@/stores/uploadStore'
import { formatBytes, formatDate } from '@/utils/format'
import VideoThumbnail from '@/components/VideoThumbnail'
import type { Video } from '@/types'

const { Title, Text } = Typography

const DOGE_PLAYER_SCRIPT = 'https://player.dogecloud.com/js/loader'
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
    default?: DogePlayerConstructor
}

function resolveDogePlayer() {
    const playerWindow = window as DogePlayerWindow
    return playerWindow.DogePlayer || playerWindow.DogeCloudPlayer || playerWindow.default || null
}

function waitForDogePlayer(timeoutMs = 5000) {
    const start = Date.now()
    return new Promise<void>((resolve, reject) => {
        const tick = () => {
            if (resolveDogePlayer()) {
                resolve()
                return
            }
            if (Date.now() - start >= timeoutMs) {
                reject(new Error('DogePlayer 构造器未暴露到全局对象'))
                return
            }
            window.setTimeout(tick, 100)
        }
        tick()
    })
}

function loadDogePlayerScript() {
    if (dogePlayerLoader) return dogePlayerLoader

    dogePlayerLoader = new Promise((resolve, reject) => {
        if (resolveDogePlayer()) {
            resolve()
            return
        }

        const existing = document.querySelector('script[data-doge-player-sdk="true"]') as HTMLScriptElement | null
        if (existing) {
            const finish = () => waitForDogePlayer().then(resolve).catch(reject)
            existing.addEventListener('load', finish, { once: true })
            existing.addEventListener('error', () => reject(new Error('加载 DogePlayer 脚本失败')))
            window.setTimeout(finish, 0)
            return
        }

        const script = document.createElement('script')
        script.src = DOGE_PLAYER_SCRIPT
        script.async = true
        script.crossOrigin = 'anonymous'
        script.setAttribute('data-doge-player-sdk', 'true')
        script.onload = () => { waitForDogePlayer().then(resolve).catch(reject) }
        script.onerror = () => reject(new Error('加载 DogePlayer 脚本失败'))
        document.head.appendChild(script)
    })

    return dogePlayerLoader
}

export default function VideoPage() {
    const { videos, total, loading, page, pageSize, fetchVideos, toggleStatus, deleteVideo, batchDelete, setPage } = useVideoStore()
    const addVideoFiles = useUploadStore((state) => state.addVideoFiles)
    const fileInputRef = useRef<HTMLInputElement>(null)
    const playerContainerRef = useRef<HTMLDivElement>(null)
    const { message, modal } = App.useApp()
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
    const [playInfo, setPlayInfo] = useState<VideoPlayInfo | null>(null)
    const [sdkError, setSdkError] = useState<string | null>(null)
    const [sdkLoading, setSdkLoading] = useState(false)
    const sdkRequirementError = useMemo(() => {
        if (!playInfo?.ready) return null

        const vcode = playInfo.vcode?.trim()
        const sdkUserID = firstNonEmptyString(
            playInfo.player_user_id,
            playerUserIDFromPlayURL(playInfo.play_url),
        )
        const sdkUserIDNum = Number(sdkUserID)

        if (!vcode || !sdkUserID || Number.isNaN(sdkUserIDNum) || sdkUserIDNum <= 0) {
            return 'DogePlayer 缺少固定 userId，请在服务端配置 IDEASAVER_DOGE_USER_ID'
        }
        return null
    }, [playInfo])

    useEffect(() => { fetchVideos(1) }, [fetchVideos])

    useEffect(() => {
        if (!playInfo || !playInfo.ready) return

        const vcode = playInfo.vcode?.trim()
        const sdkUserID = firstNonEmptyString(
            playInfo.player_user_id,
            playerUserIDFromPlayURL(playInfo.play_url),
        )
        const sdkUserIDNum = Number(sdkUserID)
        const playerContainer = playerContainerRef.current

        if (sdkRequirementError || !vcode || !sdkUserID || Number.isNaN(sdkUserIDNum) || sdkUserIDNum <= 0) {
            return
        }

        let disposed = false
        let playerInstance: DogePlayerInstance | null = null

        if (playerContainer) {
            playerContainer.innerHTML = ''
        }

        loadDogePlayerScript()
            .then(() => {
                if (disposed) return

                const DogePlayer = resolveDogePlayer()
                if (!DogePlayer || !playerContainer) {
                    setSdkError('DogePlayer SDK 未成功注入，请检查 player.dogecloud.com 的网络连通性')
                    setSdkLoading(false)
                    return
                }

                try {
                    playerInstance = new DogePlayer({
                        container: playerContainer,
                        vcode,
                        userId: sdkUserIDNum,
                        autoPlay: true,
                    })
                    setSdkLoading(false)
                } catch {
                    setSdkError('DogePlayer 初始化失败，请检查 VCode 与多吉云用户 ID 配置')
                    setSdkLoading(false)
                }
            })
            .catch(() => {
                if (!disposed) {
                    setSdkError('DogePlayer 脚本加载失败。官方播放器不支持本地部署，请确保可访问 player.dogecloud.com')
                    setSdkLoading(false)
                }
            })

        return () => {
            disposed = true
            setSdkLoading(false)
            if (playerInstance && typeof playerInstance.destroy === 'function') {
                playerInstance.destroy()
            }
            if (playerContainer) {
                playerContainer.innerHTML = ''
            }
        }
    }, [playInfo, sdkRequirementError])

    const handleUpload = () => fileInputRef.current?.click()

    const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = Array.from(e.target.files || [])
        if (files.length === 0) return

        addVideoFiles(files)
        message.success(`已将 ${files.length} 个视频加入上传队列`)
        e.target.value = ''
    }

    const handlePlay = async (id: string) => {
        try {
            setSdkError(null)
            setSdkLoading(false)
            const info = await getPlayInfo(id)
            if (!info.ready) {
                setSdkLoading(false)
                message.info(info.message || '视频仍在转码处理中，请稍后重试')
            } else {
                setSdkLoading(true)
            }
            setPlayInfo(info)
        } catch (error: unknown) {
            setSdkLoading(false)
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
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    VCode: {video.vcode || '-'}
                                                </Text>
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

            <input ref={fileInputRef} type="file" accept="video/*" multiple style={{ display: 'none' }} onChange={handleFileSelect} />

            <Modal
                title="视频播放"
                open={!!playInfo}
                onCancel={() => {
                    setPlayInfo(null)
                    setSdkError(null)
                    setSdkLoading(false)
                }}
                footer={null}
                width={820}
                destroyOnClose
            >
                {playInfo && (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                        {!playInfo.ready && (
                            <Alert
                                type="info"
                                showIcon
                                message={playInfo.message || '视频转码中，请等待多吉云回调完成后再播放'}
                            />
                        )}
                        {playInfo.ready && (sdkRequirementError || sdkError) && (
                            <Alert
                                type="error"
                                showIcon
                                message={sdkRequirementError || sdkError}
                            />
                        )}
                        {playInfo.ready ? (
                            (sdkRequirementError || sdkError) ? (
                                <div style={{ width: '100%', minHeight: 420, background: '#0b1220', color: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, textAlign: 'center' }}>
                                    官方 DogePlayer 未能启动，当前已停止回退到原生直链播放器。
                                </div>
                        ) : (
                            <div style={{ position: 'relative' }}>
                                <div
                                    ref={playerContainerRef}
                                    style={{ width: '100%', minHeight: 420, background: '#000' }}
                                />
                                {sdkLoading && (
                                    <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
                                        <Space direction="vertical" size={12} align="center">
                                            <Spin />
                                            <Text style={{ color: '#fff' }}>正在加载 DogePlayer...</Text>
                                        </Space>
                                    </div>
                                )}
                            </div>
                        )
                        ) : (
                            <div style={{ width: '100%', minHeight: 420, background: '#0b1220', color: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, textAlign: 'center' }}>
                                视频转码中，收到多吉云 `msg=transcode` 回调后会自动允许播放。
                            </div>
                        )}
                    </Space>
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
