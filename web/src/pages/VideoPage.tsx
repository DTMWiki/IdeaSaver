import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Card, Row, Col, Button, Space, Typography, Switch, Popconfirm, Empty, Spin, Modal, App, Pagination, Alert, Tag, Input, Grid } from 'antd'
import {
    UploadOutlined,
    PlayCircleOutlined,
    DeleteOutlined,
    ShareAltOutlined,
    CopyOutlined,
} from '@ant-design/icons'
import { useVideoStore } from '@/stores/videoStore'
import { getPlayInfo, type VideoPlayInfo } from '@/api/videos'
import { useUploadStore } from '@/stores/uploadStore'
import { formatBytes, formatDate, copyToClipboard } from '@/utils/format'
import {
    buildDogeIframeCode,
    firstNonEmptyString,
    loadDogePlayerScript,
    normalizeDimension,
    playerUserIDFromPlayURL,
    resolveDogePlayer,
    type DogePlayerInstance,
} from '@/utils/dogePlayer'
import VideoThumbnail from '@/components/VideoThumbnail'
import type { Video } from '@/types'

const { Title, Text } = Typography

export default function VideoPage() {
    const { videos, total, loading, page, pageSize, fetchVideos, toggleStatus, deleteVideo, batchDelete, setPage } = useVideoStore()
    const addVideoFiles = useUploadStore((state) => state.addVideoFiles)
    const fileInputRef = useRef<HTMLInputElement>(null)
    const playerContainerRef = useRef<HTMLDivElement | null>(null)
    const [playerContainerTick, setPlayerContainerTick] = useState(0)
    const { message, modal } = App.useApp()
    const screens = Grid.useBreakpoint()
    const isMobile = !screens.md
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
    const [playInfo, setPlayInfo] = useState<VideoPlayInfo | null>(null)
    const [sdkError, setSdkError] = useState<string | null>(null)
    const [sdkLoading, setSdkLoading] = useState(false)
    const [shareVideo, setShareVideo] = useState<Video | null>(null)
    const [shareLoading, setShareLoading] = useState(false)
    const [shareContext, setShareContext] = useState<{ vcode: string; userId: string } | null>(null)
    const [shareMessage, setShareMessage] = useState('')
    const [shareAutoPlay, setShareAutoPlay] = useState(false)
    const [shareWidth, setShareWidth] = useState('')
    const [shareHeight, setShareHeight] = useState('')
    const sdkRequirementError = useMemo(() => {
        if (!playInfo?.ready) return null

        const vcode = playInfo.vcode?.trim()
        const sdkUserID = firstNonEmptyString(
            playInfo.player_user_id,
            playerUserIDFromPlayURL(playInfo.play_url),
        )
        const sdkUserIDNum = Number(sdkUserID)

        if (!vcode || !sdkUserID || Number.isNaN(sdkUserIDNum) || sdkUserIDNum <= 0) {
            return '播放器配置异常，请联系管理员'
        }
        return null
    }, [playInfo])
    const shareIframeCode = useMemo(() => {
        if (!shareContext) return ''
        return buildDogeIframeCode({
            vcode: shareContext.vcode,
            userId: shareContext.userId,
            autoPlay: shareAutoPlay,
            width: normalizeDimension(shareWidth, '600'),
            height: normalizeDimension(shareHeight, '400'),
        })
    }, [shareAutoPlay, shareContext, shareHeight, shareWidth])

    const attachPlayerContainer = useCallback((node: HTMLDivElement | null) => {
        playerContainerRef.current = node
        setPlayerContainerTick((value) => value + 1)
    }, [])

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

        if (!playerContainer) {
            return
        }

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
                if (!DogePlayer) {
                    setSdkError('播放器加载失败，请稍后重试')
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
                    setSdkError('播放器初始化失败，请稍后重试或联系管理员')
                    setSdkLoading(false)
                }
            })
            .catch(() => {
                if (!disposed) {
                    setSdkError('无法加载播放器，请检查网络后重试')
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
    }, [playInfo, playerContainerTick, sdkRequirementError])

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

    const handleShare = async (video: Video) => {
        setShareVideo(video)
        setShareLoading(true)
        setShareContext(null)
        setShareMessage('')
        setShareAutoPlay(false)
        setShareWidth('')
        setShareHeight('')

        try {
            const info = await getPlayInfo(video.id)
            if (!info.ready) {
                setShareMessage(info.message || '视频仍在转码中，暂时不能生成分享链接')
                return
            }

            const vcode = firstNonEmptyString(info.vcode, video.vcode)
            const playerUserId = firstNonEmptyString(
                info.player_user_id,
                video.player_user_id,
                playerUserIDFromPlayURL(info.play_url),
                playerUserIDFromPlayURL(video.play_url),
            )
            if (!vcode || !playerUserId) {
                setShareMessage('当前视频尚未就绪，暂时无法生成分享链接')
                return
            }

            setShareVideo({ ...video, play_count: info.play_count ?? video.play_count })
            setShareContext({ vcode, userId: playerUserId })
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            setShareMessage(maybeMessage || '生成分享链接失败')
        } finally {
            setShareLoading(false)
        }
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
                                        <Button type="text" icon={<ShareAltOutlined />} onClick={(e) => { e.stopPropagation(); handleShare(video) }} key="share">
                                            分享
                                        </Button>,
                                        <Button type="text" icon={<PlayCircleOutlined />} onClick={(e) => { e.stopPropagation(); handlePlay(video.id) }} key="play">
                                            {!isMobile ? '播放' : null}
                                        </Button>,
                                        <Popconfirm
                                            key="del"
                                            title="删除此视频？"
                                            onConfirm={(e) => { e?.stopPropagation(); deleteVideo(video.id) }}
                                            onCancel={(e) => e?.stopPropagation()}
                                        >
                                            <Button type="text" danger icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()}>
                                                {!isMobile ? '删除' : null}
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
                                                    播放码: {video.vcode || '-'}
                                                </Text>
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    播放次数: {video.play_count ?? 0}
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
                width={isMobile ? 'calc(100vw - 24px)' : 820}
                destroyOnClose
            >
                {playInfo && (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                        {!playInfo.ready && (
                            <Alert
                                type="info"
                                showIcon
                                message={playInfo.message || '视频转码中，请稍后再试'}
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
                                    播放器未能启动，请稍后重试或联系管理员。
                                </div>
                        ) : (
                            <div style={{ position: 'relative' }}>
                                <div
                                    ref={attachPlayerContainer}
                                    style={{ width: '100%', minHeight: 420, background: '#000' }}
                                />
                                {sdkLoading && (
                                    <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
                                        <Space direction="vertical" size={12} align="center">
                                            <Spin />
                                            <Text style={{ color: '#fff' }}>正在加载播放器...</Text>
                                        </Space>
                                    </div>
                                )}
                            </div>
                        )
                        ) : (
                            <div style={{ width: '100%', minHeight: 420, background: '#0b1220', color: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, textAlign: 'center' }}>
                                视频转码中，完成后即可播放。
                            </div>
                        )}
                    </Space>
                )}
            </Modal>

            <Modal
                title={shareVideo ? `分享视频：${shareVideo.title}` : '分享视频'}
                open={!!shareVideo}
                onCancel={() => {
                    setShareVideo(null)
                    setShareLoading(false)
                    setShareContext(null)
                    setShareMessage('')
                    setShareAutoPlay(false)
                    setShareWidth('')
                    setShareHeight('')
                }}
                footer={null}
                width={isMobile ? 'calc(100vw - 24px)' : 700}
                destroyOnClose
            >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Alert
                        type={shareContext ? 'success' : 'info'}
                        showIcon
                        message="使用嵌入代码接入播放器"
                        description="你已开启防盗链时，不建议再暴露直接访问地址。这里默认只提供可复制的 iframe 嵌入代码。"
                    />
                    <div>
                        <Text strong>使用方法（Markdown / HTML 嵌入）</Text>
                        <ol style={{ margin: '8px 0 0', paddingInlineStart: 18, color: 'rgba(71,85,105,0.92)' }}>
                            <li>复制下方 iframe 代码，粘贴到支持 HTML 的页面中。</li>
                            <li>若系统支持 Markdown 中嵌入 HTML，可直接使用这段代码。</li>
                            <li>如果视频仍在转码，请等待完成后再复制。</li>
                        </ol>
                    </div>
                    {shareLoading ? (
                        <Alert type="info" showIcon message="正在生成嵌入代码..." />
                    ) : shareContext ? (
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                            <Text type="secondary">播放次数：{shareVideo?.play_count ?? 0}</Text>
                            <Space wrap size={12} style={{ width: '100%' }}>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">自动播放</Text>
                                    <Switch checked={shareAutoPlay} onChange={setShareAutoPlay} />
                                </div>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">宽度</Text>
                                    <Input
                                        value={shareWidth}
                                        onChange={(e) => setShareWidth(e.target.value.replace(/[^\d]/g, ''))}
                                        placeholder="600"
                                        style={{ width: isMobile ? '100%' : 120 }}
                                    />
                                </div>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">高度</Text>
                                    <Input
                                        value={shareHeight}
                                        onChange={(e) => setShareHeight(e.target.value.replace(/[^\d]/g, ''))}
                                        placeholder="400"
                                        style={{ width: isMobile ? '100%' : 120 }}
                                    />
                                </div>
                            </Space>
                            <div style={{ position: 'relative', borderRadius: 14, overflow: 'hidden', background: '#0f172a' }}>
                                <Button
                                    type="primary"
                                    icon={<CopyOutlined />}
                                    size="small"
                                    style={{ position: 'absolute', top: 12, right: 12, zIndex: 1 }}
                                    onClick={async () => {
                                        await copyToClipboard(shareIframeCode)
                                        message.success('嵌入代码已复制')
                                    }}
                                >
                                    复制
                                </Button>
                                <pre style={{ margin: 0, padding: '52px 16px 16px', color: '#e2e8f0', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13, lineHeight: 1.6 }}>
                                    <code>{shareIframeCode}</code>
                                </pre>
                            </div>
                        </Space>
                    ) : (
                        <Alert type="warning" showIcon message={shareMessage || '当前视频暂不可分享'} />
                    )}
                </Space>
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
