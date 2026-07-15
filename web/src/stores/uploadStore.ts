import { create } from 'zustand'
import type { UploadFileTask } from '@/types'
import * as uploadApi from '@/api/upload'
import * as videosApi from '@/api/videos'
import { useFileStore } from '@/stores/fileStore'
import { useVideoStore } from '@/stores/videoStore'

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB
const MAX_CONCURRENT = 3

interface UploadState {
    tasks: UploadFileTask[]
    panelVisible: boolean

    setPanelVisible: (visible: boolean) => void
    addFiles: (files: File[], parentId: string | null) => void
    addVideoFiles: (files: File[]) => void
    syncVideoTask: (videoId: string, transcodeStatus: string, message?: string, vcode?: string) => void
    pauseTask: (id: string) => void
    resumeTask: (id: string) => void
    removeTask: (id: string) => void
    clearCompleted: () => void
}

let taskCounter = 0
let activeCount = 0

function generateLocalId(): string {
    return `local-${Date.now()}-${taskCounter++}`
}

function buildTask(file: File, targetType: 'file' | 'video', parentId: string | null): UploadFileTask {
    const totalChunks = targetType === 'file'
        ? Math.max(1, Math.ceil(file.size / CHUNK_SIZE))
        : 1

    return {
        id: generateLocalId(),
        file,
        parentId,
        targetType,
        filename: file.name,
        totalSize: file.size,
        uploadedSize: 0,
        chunkSize: targetType === 'file' ? CHUNK_SIZE : file.size,
        totalChunks,
        uploadedChunks: 0,
        status: 'pending',
        progress: 0,
        speed: 0,
        phase: 'uploading',
        detail: targetType === 'video' ? '等待开始上传到业务服务器' : undefined,
    }
}

export const useUploadStore = create<UploadState>((set, get) => ({
    tasks: [],
    panelVisible: false,

    setPanelVisible: (visible: boolean) => set({ panelVisible: visible }),

    addFiles: (files: File[], parentId: string | null) => {
        const newTasks = files.map((file) => buildTask(file, 'file', parentId))
        set((state) => ({
            tasks: [...state.tasks, ...newTasks],
            panelVisible: true,
        }))
        processQueue()
    },

    addVideoFiles: (files: File[]) => {
        const newTasks = files.map((file) => buildTask(file, 'video', null))
        set((state) => ({
            tasks: [...state.tasks, ...newTasks],
            panelVisible: true,
        }))
        processQueue()
    },

    syncVideoTask: (videoId: string, transcodeStatus: string, message?: string, vcode?: string) => {
        set((state) => ({
            tasks: state.tasks.map((task) => {
                if (task.videoId !== videoId) return task

                if (transcodeStatus === 'ready') {
                    return {
                        ...task,
                        vcode: vcode || task.vcode,
                        phase: undefined,
                        detail: message || '视频转码完成，可开始播放',
                    }
                }

                if (transcodeStatus === 'failed' || transcodeStatus === 'blocked') {
                    return {
                        ...task,
                        vcode: vcode || task.vcode,
                        status: 'failed' as const,
                        error: message || '视频暂不可播放',
                        detail: message || '视频暂不可播放',
                    }
                }

                return {
                    ...task,
                    vcode: vcode || task.vcode,
                    detail: message || task.detail,
                }
            }),
        }))
    },

    pauseTask: (id: string) => {
        const task = get().tasks.find((item) => item.id === id)
        if (!task || task.targetType !== 'file') return

        set((state) => ({
            tasks: state.tasks.map((item) =>
                item.id === id && item.status === 'uploading'
                    ? { ...item, status: 'paused' as const }
                    : item,
            ),
        }))

        if (task.taskId) {
            uploadApi.pauseUpload(task.taskId).catch(() => { })
        }
    },

    resumeTask: (id: string) => {
        const task = get().tasks.find((item) => item.id === id)
        if (!task || task.targetType !== 'file') return

        set((state) => ({
            tasks: state.tasks.map((item) =>
                item.id === id && item.status === 'paused'
                    ? { ...item, status: 'pending' as const }
                    : item,
            ),
        }))

        if (task.taskId) {
            uploadApi.resumeUpload(task.taskId).catch(() => { })
        }

        processQueue()
    },

    removeTask: (id: string) => {
        set((state) => ({
            tasks: state.tasks.filter((item) => item.id !== id),
        }))
    },

    clearCompleted: () => {
        set((state) => ({
            tasks: state.tasks.filter((item) => item.status !== 'completed' && item.status !== 'failed'),
        }))
    },
}))

async function processQueue() {
    const pending = useUploadStore.getState().tasks.filter((task) => task.status === 'pending')

    while (activeCount < MAX_CONCURRENT && pending.length > 0) {
        const task = pending.shift()
        if (!task) break

        activeCount++
        processTask(task.id).finally(() => {
            activeCount--
            processQueue()
        })
    }
}

async function processTask(localId: string) {
    const task = useUploadStore.getState().tasks.find((item) => item.id === localId)
    if (!task || task.status !== 'pending') return

    updateTask(localId, { status: 'uploading', error: undefined })

    try {
        if (task.targetType === 'video') {
            await processVideoTask(task)
        } else {
            await processFileTask(task)
        }
    } catch (err) {
        updateTask(localId, {
            status: 'failed',
            speed: 0,
            error: extractErrorMessage(err),
        })
    }
}

async function processFileTask(task: UploadFileTask) {
    const localId = task.id
    let taskId = task.taskId
    const startChunk = task.uploadedChunks

    if (!taskId) {
        const resp = await uploadApi.initUpload({
            filename: task.filename,
            size: task.totalSize,
            mime_type: task.file.type || 'application/octet-stream',
            parent_id: task.parentId || undefined,
        })
        taskId = resp.task_id
        updateTask(localId, { taskId })
    }

    const startTime = Date.now()
    let uploadedBytes = task.uploadedSize

    for (let i = startChunk; i < task.totalChunks; i++) {
        const current = useUploadStore.getState().tasks.find((item) => item.id === localId)
        if (!current || current.status === 'paused') return

        const start = i * task.chunkSize
        const end = Math.min(start + task.chunkSize, task.totalSize)
        const chunk = task.file.slice(start, end)

        await uploadApi.uploadChunk(taskId, i, chunk, (loaded) => {
            const currentUploaded = Math.min(uploadedBytes + loaded, task.totalSize)
            const elapsed = (Date.now() - startTime) / 1000
            const speed = elapsed > 0 ? currentUploaded / elapsed : 0
            updateTask(localId, {
                uploadedSize: currentUploaded,
                progress: Math.round((currentUploaded / task.totalSize) * 100),
                speed,
                phase: 'uploading',
            })
        })

        uploadedBytes = Math.min((i + 1) * task.chunkSize, task.totalSize)

        updateTask(localId, {
            uploadedChunks: i + 1,
            uploadedSize: uploadedBytes,
            progress: Math.round((uploadedBytes / task.totalSize) * 100),
            phase: 'uploading',
        })
    }

    const result = await uploadApi.completeUpload(taskId, task.parentId)

    updateTask(localId, {
        status: 'completed',
        progress: 100,
        speed: 0,
        uploadedSize: task.totalSize,
        phase: 'processing',
        url: result.url,
        markdown: result.markdown,
    })

    useFileStore.getState().refresh().catch(() => { })
}

async function processVideoTask(task: UploadFileTask) {
    const localId = task.id
    const startTime = Date.now()

    const video = await videosApi.uploadVideo(task.file, undefined, {
        onProgress: (progress) => {
            const uploadedSize = Math.min(
                task.totalSize,
                Math.round((task.totalSize * progress.percent) / 100),
            )
            const elapsed = (Date.now() - startTime) / 1000
            const speed = elapsed > 0 ? uploadedSize / elapsed : 0
            updateTask(localId, {
                uploadedSize,
                uploadedChunks: progress.percent >= 100 ? 1 : 0,
                progress: progress.percent,
                speed,
                phase: progress.phase === 'processing' ? 'processing' : 'uploading',
                detail: progress.phase === 'processing'
                    ? '文件已送达，等待转码任务创建'
                    : '正在上传',
            })
        },
    })

    updateTask(localId, {
        status: 'completed',
        progress: 100,
        speed: 0,
        uploadedSize: task.totalSize,
        uploadedChunks: 1,
        phase: 'waiting_transcode',
        detail: '视频已上传，转码完成后可播放',
        videoId: video.id,
        vcode: video.vcode || undefined,
    })

    useVideoStore.getState().fetchVideos(1).catch(() => { })
}

function updateTask(localId: string, updates: Partial<UploadFileTask>) {
    useUploadStore.setState((state) => ({
        tasks: state.tasks.map((task) => (task.id === localId ? { ...task, ...updates } : task)),
    }))
}

function extractErrorMessage(error: unknown): string {
    const responseMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    if (responseMessage) {
        return responseMessage
    }
    if (error instanceof Error) {
        return error.message
    }
    return '上传失败'
}
