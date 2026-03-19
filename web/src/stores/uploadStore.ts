import { create } from 'zustand'
import type { UploadFileTask } from '@/types'
import * as uploadApi from '@/api/upload'

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB
const MAX_CONCURRENT = 3

interface UploadState {
    tasks: UploadFileTask[]
    panelVisible: boolean

    setPanelVisible: (visible: boolean) => void
    addFiles: (files: File[], parentId: string | null) => void
    pauseTask: (id: string) => void
    resumeTask: (id: string) => void
    removeTask: (id: string) => void
    clearCompleted: () => void
}

let taskCounter = 0

function generateLocalId(): string {
    return `local-${Date.now()}-${taskCounter++}`
}

export const useUploadStore = create<UploadState>((set, get) => ({
    tasks: [],
    panelVisible: false,

    setPanelVisible: (visible: boolean) => set({ panelVisible: visible }),

    addFiles: (files: File[], parentId: string | null) => {
        const newTasks: UploadFileTask[] = files.map((file) => {
            const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
            return {
                id: generateLocalId(),
                file,
                parentId,
                filename: file.name,
                totalSize: file.size,
                uploadedSize: 0,
                chunkSize: CHUNK_SIZE,
                totalChunks,
                uploadedChunks: 0,
                status: 'pending',
                progress: 0,
                speed: 0,
            }
        })

        set((state) => ({
            tasks: [...state.tasks, ...newTasks],
            panelVisible: true,
        }))

        // Start processing queue
        processQueue()
    },

    pauseTask: (id: string) => {
        set((state) => ({
            tasks: state.tasks.map((t) =>
                t.id === id && t.status === 'uploading' ? { ...t, status: 'paused' as const } : t,
            ),
        }))
        // If server task exists, notify server
        const task = get().tasks.find((t) => t.id === id)
        if (task?.taskId) {
            uploadApi.pauseUpload(task.taskId).catch(() => { })
        }
    },

    resumeTask: (id: string) => {
        set((state) => ({
            tasks: state.tasks.map((t) =>
                t.id === id && t.status === 'paused' ? { ...t, status: 'pending' as const } : t,
            ),
        }))
        processQueue()
    },

    removeTask: (id: string) => {
        set((state) => ({
            tasks: state.tasks.filter((t) => t.id !== id),
        }))
    },

    clearCompleted: () => {
        set((state) => ({
            tasks: state.tasks.filter((t) => t.status !== 'completed' && t.status !== 'failed'),
        }))
    },
}))

// ---- Upload Queue Processor ----

let activeCount = 0

async function processQueue() {
    const { tasks } = useUploadStore.getState()
    const pending = tasks.filter((t) => t.status === 'pending')

    while (activeCount < MAX_CONCURRENT && pending.length > 0) {
        const task = pending.shift()!
        activeCount++
        processTask(task.id).finally(() => {
            activeCount--
            processQueue()
        })
    }
}

async function processTask(localId: string) {
    const store = useUploadStore
    const task = store.getState().tasks.find((t) => t.id === localId)
    if (!task || task.status !== 'pending') return

    // Update status to uploading
    updateTask(localId, { status: 'uploading' })

    try {
        // Step 1: Init upload on server
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

        // Step 2: Upload chunks
        const startTime = Date.now()
        let uploadedBytes = task.uploadedSize

        for (let i = startChunk; i < task.totalChunks; i++) {
            // Check if paused
            const current = store.getState().tasks.find((t) => t.id === localId)
            if (!current || current.status === 'paused') return

            const start = i * task.chunkSize
            const end = Math.min(start + task.chunkSize, task.totalSize)
            const chunk = task.file.slice(start, end)

            await uploadApi.uploadChunk(taskId!, i, chunk, (loaded) => {
                const currentUploaded = uploadedBytes + loaded
                const elapsed = (Date.now() - startTime) / 1000
                const speed = elapsed > 0 ? currentUploaded / elapsed : 0
                updateTask(localId, {
                    uploadedSize: currentUploaded,
                    progress: Math.round((currentUploaded / task.totalSize) * 100),
                    speed,
                })
            })

            uploadedBytes = (i + 1) * task.chunkSize
            if (uploadedBytes > task.totalSize) uploadedBytes = task.totalSize

            updateTask(localId, {
                uploadedChunks: i + 1,
                uploadedSize: uploadedBytes,
                progress: Math.round((uploadedBytes / task.totalSize) * 100),
            })
        }

        // Step 3: Complete upload
        const result = await uploadApi.completeUpload(taskId!, task.parentId)
        updateTask(localId, {
            status: 'completed',
            progress: 100,
            speed: 0,
            url: result.url,
            markdown: result.markdown,
        })
    } catch (err) {
        const message = err instanceof Error ? err.message : '上传失败'
        updateTask(localId, { status: 'failed', speed: 0, error: message })
    }
}

function updateTask(localId: string, updates: Partial<UploadFileTask>) {
    useUploadStore.setState((state) => ({
        tasks: state.tasks.map((t) => (t.id === localId ? { ...t, ...updates } : t)),
    }))
}
