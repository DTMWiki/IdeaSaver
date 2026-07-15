import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Card, Row, Col, Button, Space, Typography, Switch, Popconfirm, Empty, Spin, Modal, App, Pagination, Alert, Tag, Input, Grid, Checkbox, Dropdown } from 'antd'
import type { MenuProps } from 'antd'
import {
    UploadOutlined,
    PlayCircleOutlined,
    DeleteOutlined,
    CodeOutlined,
    CopyOutlined,
    MoreOutlined,
} from '@ant-design/icons'
import { isPendingTranscode, useVideoStore } from '@/stores/videoStore'
import { getPlayInfo, type VideoPlayInfo } from '@/api/videos'
import { useUploadStore } from '@/stores/uploadStore'
import { formatBytes, formatDate, copyToClipboard } from '@/utils/format'
import {
    buildDogeIframeCode,
    buildDogeShareURL,
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
    const syncVideoTask = useUploadStore((state) => state.syncVideoTask)
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
    const [embedVideo, setEmbedVideo] = useState<Video | null>(null)
    const [embedLoading, setEmbedLoading] = useState(false)
    const [embedContext, setEmbedContext] = useState<{ vcode: string; userId: string } | null>(null)
    const [embedMessage, setEmbedMessage] = useState('')
    const [embedAutoPlay, setEmbedAutoPlay] = useState(false)
    const [embedWidth, setEmbedWidth] = useState('')
    const [embedHeight, setEmbedHeight] = useState('')
    const hasPendingTranscode = useMemo(
        () => videos.some((v) => isPendingTranscode(v.transcode_status)),
        [videos],
    )
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
    const embedIframeCode = useMemo(() => {
        if (!embedContext) return ''
        return buildDogeIframeCode({
            vcode: embedContext.vcode,
            userId: embedContext.userId,
            autoPlay: embedAutoPlay,
            width: normalizeDimension(embedWidth, '600'),
            height: normalizeDimension(embedHeight, '400'),
        })
    }, [embedAutoPlay, embedContext, embedHeight, embedWidth])

    const attachPlayerContainer = useCallback((node: HTMLDivElement | null) => {
        playerContainerRef.current = node
        setPlayerContainerTick((value) => value + 1)
    }, [])

    useEffect(() => { void fetchVideos(1) }, [fetchVideos])

    // Poll while any video on the page is still transcoding (silent refresh).
    useEffect(() => {
        if (!hasPendingTranscode) return
        const timer = window.setInterval(() => {
            void fetchVideos(undefined, { silent: true }).then(() => {
                const latest = useVideoStore.getState().videos
                for (const video of latest) {
                    if (video.transcode_status === 'ready' || video.transcode_status === 'failed' || video.transcode_status === 'blocked') {
                        syncVideoTask(video.id, video.transcode_status, video.transcode_message, video.vcode)
                    }
                }
            })
        }, 5000)
        return () => window.clearInterval(timer)
    }, [hasPendingTranscode, fetchVideos, syncVideoTask])

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
            setSdkLoading(true)
            const info = await getPlayInfo(id)
            // Open modal as the single feedback surface (avoid toast + modal double noise).
            setPlayInfo(info)
            if (!info.ready) {
                setSdkLoading(false)
            }
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

    const handleEmbed = async (video: Video) => {
        setEmbedVideo(video)
        setEmbedLoading(true)
        setEmbedContext(null)
        setEmbedMessage('')
        setEmbedAutoPlay(false)
        setEmbedWidth('')
        setEmbedHeight('')

        try {
            const info = await getPlayInfo(video.id)
            if (!info.ready) {
                setEmbedMessage(info.message || '视频仍在转码中，暂时不能生成嵌入代码')
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
                setEmbedMessage('当前视频尚未就绪，暂时无法生成嵌入代码')
                return
            }

            setEmbedVideo({ ...video, play_count: info.play_count ?? video.play_count })
            setEmbedContext({ vcode, userId: playerUserId })
        } catch (error: unknown) {
            const maybeMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
            setEmbedMessage(maybeMessage || '生成嵌入代码失败')
        } finally {
            setEmbedLoading(false)
        }
    }

    const videoCardActions = (video: Video) => {
        if (isMobile) {
            const items: MenuProps['items'] = [
                {
                    key: 'play',
                    icon: <PlayCircleOutlined />,
                    label: '播放',
                    onClick: () => { void handlePlay(video.id) },
                },
                {
                    key: 'embed',
                    icon: <CodeOutlined />,
                    label: '网页嵌入',
                    onClick: () => { void handleEmbed(video) },
                },
                {
                    key: 'delete',
                    icon: <DeleteOutlined />,
                    label: '删除',
                    danger: true,
                    onClick: () => {
                        modal.confirm({
                            title: '删除此视频？',
                            content: '删除后不可恢复，并释放占用的存储配额',
                            okText: '删除',
                            okType: 'danger',
                            cancelText: '取消',
                            onOk: () => deleteVideo(video.id),
                        })
                    },
                },
            ]
            return [
                <Dropdown key="more" menu={{ items }} trigger={['click']}>
                    <Button type="text" icon={<MoreOutlined />}>
                        更多
                    </Button>
                </Dropdown>,
            ]
        }

        return [
            <Button type="text" icon={<CodeOutlined />} onClick={() => { void handleEmbed(video) }} key="embed">
                嵌入
            </Button>,
            <Button type="text" icon={<PlayCircleOutlined />} onClick={() => { void handlePlay(video.id) }} key="play">
                播放
            </Button>,
            <Popconfirm
                key="del"
                title="删除此视频？"
                description="删除后不可恢复，并释放占用的存储配额"
                onConfirm={() => { void deleteVideo(video.id) }}
            >
                <Button type="text" danger icon={<DeleteOutlined />}>
                    删除
                </Button>
            </Popconfirm>,
        ]
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
                <Empty description="还没有视频">
                    <Button type="primary" icon={<UploadOutlined />} onClick={handleUpload}>
                        上传第一个视频
                    </Button>
                </Empty>
            ) : (
                <>
                    <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
                        点击封面即可播放；批量操作请勾选右上角。转码中的视频会自动刷新状态。
                        {hasPendingTranscode && (
                            <Tag color="processing" style={{ marginLeft: 8 }}>转码状态自动更新中</Tag>
                        )}
                    </Text>
                    <Row gutter={[16, 16]}>
                        {videos.map((video) => (
                            <Col xs={24} sm={12} lg={8} xl={6} key={video.id}>
                                <Card
                                    hoverable
                                    style={{ borderColor: selectedIds.has(video.id) ? '#1677ff' : undefined }}
                                    cover={(
                                        <div style={{ position: 'relative' }}>
                                            <div
                                                role="button"
                                                tabIndex={0}
                                                onClick={() => { void handlePlay(video.id) }}
                                                onKeyDown={(e) => {
                                                    if (e.key === 'Enter' || e.key === ' ') {
                                                        e.preventDefault()
                                                        void handlePlay(video.id)
                                                    }
                                                }}
                                                style={{ cursor: 'pointer' }}
                                            >
                                                <VideoThumbnail
                                                    src={video.thumbnail_small_url || video.thumbnail_url}
                                                    alt={video.title}
                                                    height={160}
                                                    borderRadius={0}
                                                    iconSize={30}
                                                />
                                            </div>
                                            <div
                                                style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}
                                                onClick={(e) => e.stopPropagation()}
                                            >
                                                <Checkbox
                                                    checked={selectedIds.has(video.id)}
                                                    onChange={() => toggleSelect(video.id)}
                                                    style={{ background: 'rgba(255,255,255,0.9)', borderRadius: 4, padding: '2px 4px' }}
                                                />
                                            </div>
                                        </div>
                                    )}
                                    actions={videoCardActions(video)}
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
                                // Official DogeCloud web player as fallback (docs: player.html?vcode&userId)
                                (() => {
                                    const userId = firstNonEmptyString(
                                        playInfo.player_user_id,
                                        playerUserIDFromPlayURL(playInfo.play_url),
                                    )
                                    const src = buildDogeShareURL(playInfo.vcode || '', userId, { autoPlay: true })
                                    if (!src) {
                                        return (
                                            <div style={{ width: '100%', minHeight: 280, background: '#0b1220', color: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, textAlign: 'center' }}>
                                                {sdkRequirementError || sdkError || '播放器未能启动，请稍后重试'}
                                            </div>
                                        )
                                    }
                                    return (
                                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                                            <Alert type="warning" showIcon message="本地播放器加载失败，已切换官方网页播放器" />
                                            <iframe
                                                title="视频播放"
                                                src={src}
                                                style={{ width: '100%', minHeight: isMobile ? 240 : 420, border: 0, background: '#000' }}
                                                allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture; fullscreen"
                                                allowFullScreen
                                            />
                                        </Space>
                                    )
                                })()
                            ) : (
                                <div style={{ position: 'relative' }}>
                                    <div
                                        ref={attachPlayerContainer}
                                        style={{ width: '100%', minHeight: isMobile ? 240 : 420, background: '#000' }}
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
                            <div style={{ width: '100%', minHeight: 200, background: '#0b1220', color: '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, textAlign: 'center' }}>
                                {playInfo.message || '视频转码中，完成后即可播放。可稍后再点封面重试。'}
                            </div>
                        )}
                    </Space>
                )}
            </Modal>

            <Modal
                title={embedVideo ? `网页嵌入：${embedVideo.title}` : '网页嵌入'}
                open={!!embedVideo}
                onCancel={() => {
                    setEmbedVideo(null)
                    setEmbedLoading(false)
                    setEmbedContext(null)
                    setEmbedMessage('')
                    setEmbedAutoPlay(false)
                    setEmbedWidth('')
                    setEmbedHeight('')
                }}
                footer={null}
                width={isMobile ? 'calc(100vw - 24px)' : 700}
                destroyOnClose
            >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Alert
                        type={embedContext ? 'success' : 'info'}
                        showIcon
                        message="这是网页嵌入码，不是文件外链分享"
                        description="与「我的分享」中的文件分享不同：这里生成的是多吉云播放器 iframe，适合贴到博客/Wiki。若要发文件给他人下载，请在文件管理里创建分享链接。"
                    />
                    <div>
                        <Text strong>怎么用</Text>
                        <ol style={{ margin: '8px 0 0', paddingInlineStart: 18, color: 'rgba(71,85,105,0.92)' }}>
                            <li>复制下方 iframe 代码，贴进支持 HTML 的网页。</li>
                            <li>需要自动播放或调整尺寸时，改选项后重新复制。</li>
                            <li>转码未完成时无法生成；本页会自动刷新转码状态。</li>
                        </ol>
                    </div>
                    {embedLoading ? (
                        <Alert type="info" showIcon message="正在生成嵌入代码..." />
                    ) : embedContext ? (
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                            <Text type="secondary">播放次数：{embedVideo?.play_count ?? 0}</Text>
                            <Space wrap size={12} style={{ width: '100%' }}>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">自动播放</Text>
                                    <Switch checked={embedAutoPlay} onChange={setEmbedAutoPlay} />
                                </div>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">宽度</Text>
                                    <Input
                                        value={embedWidth}
                                        onChange={(e) => setEmbedWidth(e.target.value.replace(/[^\d]/g, ''))}
                                        placeholder="600"
                                        style={{ width: isMobile ? '100%' : 120 }}
                                    />
                                </div>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <Text type="secondary">高度</Text>
                                    <Input
                                        value={embedHeight}
                                        onChange={(e) => setEmbedHeight(e.target.value.replace(/[^\d]/g, ''))}
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
                                        await copyToClipboard(embedIframeCode)
                                        message.success('嵌入代码已复制')
                                    }}
                                >
                                    复制
                                </Button>
                                <pre style={{ margin: 0, padding: '52px 16px 16px', color: '#e2e8f0', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13, lineHeight: 1.6 }}>
                                    <code>{embedIframeCode}</code>
                                </pre>
                            </div>
                        </Space>
                    ) : (
                        <Alert type="warning" showIcon message={embedMessage || '当前视频暂不可嵌入'} />
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
