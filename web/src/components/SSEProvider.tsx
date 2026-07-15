import { useEffect, useRef, type ReactNode } from 'react'
import { App } from 'antd'
import { useAuthStore } from '@/stores/authStore'
import { useFileStore } from '@/stores/fileStore'
import { useVideoStore } from '@/stores/videoStore'
import { useUploadStore } from '@/stores/uploadStore'

interface SSEProviderProps {
    children: ReactNode
}

export default function SSEProvider({ children }: SSEProviderProps) {
    const isLoggedIn = useAuthStore((s) => s.isLoggedIn)
    const { refresh } = useFileStore()
    const refreshVideos = useVideoStore((state) => state.fetchVideos)
    const syncVideoTask = useUploadStore((state) => state.syncVideoTask)
    const { notification } = App.useApp()
    const eventSourceRef = useRef<EventSource | null>(null)
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

    useEffect(() => {
        if (!isLoggedIn) return

        function connect() {
            if (eventSourceRef.current) {
                eventSourceRef.current.close()
            }
            if (reconnectTimerRef.current) {
                clearTimeout(reconnectTimerRef.current)
                reconnectTimerRef.current = null
            }

            // Cookie session is sent automatically for same-origin EventSource.
            const es = new EventSource('/api/events')
            eventSourceRef.current = es

            es.onopen = () => {
                if (reconnectTimerRef.current) {
                    clearTimeout(reconnectTimerRef.current)
                    reconnectTimerRef.current = null
                }
            }

            es.addEventListener('upload_complete', (e) => {
                try {
                    const data = JSON.parse(e.data)
                    notification.success({
                        message: '上传完成',
                        description: `文件 "${data.filename || '未知'}" 已上传成功`,
                        duration: 5,
                    })
                    refresh()
                } catch {
                    // ignore parse errors
                }
            })

            es.addEventListener('video_transcode_complete', (e) => {
                try {
                    const data = JSON.parse(e.data)
                    syncVideoTask(data.video_id, data.transcode_status, data.transcode_message, data.vcode)
                    notification.success({
                        message: '视频转码完成',
                        description: `视频 "${data.title || '未知'}" 转码完成`,
                        duration: 5,
                    })
                    refreshVideos().catch(() => { })
                } catch {
                    // ignore
                }
            })

            es.addEventListener('video_upload_complete', () => {
                refreshVideos().catch(() => { })
            })

            es.addEventListener('video_status_update', (e) => {
                try {
                    const data = JSON.parse(e.data)
                    syncVideoTask(data.video_id, data.transcode_status, data.transcode_message, data.vcode)
                } catch {
                    // ignore
                }
                refreshVideos().catch(() => { })
            })

            es.onerror = () => {
                if (eventSourceRef.current !== es) {
                    return
                }
                es.close()
                if (reconnectTimerRef.current) {
                    clearTimeout(reconnectTimerRef.current)
                }
                reconnectTimerRef.current = setTimeout(connect, 5000)
            }
        }

        connect()

        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close()
            }
            if (reconnectTimerRef.current) {
                clearTimeout(reconnectTimerRef.current)
            }
        }
    }, [isLoggedIn, notification, refresh, refreshVideos, syncVideoTask])

    return <>{children}</>
}
