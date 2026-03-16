import client from './client'
import type { UploadTask } from '@/types'

export interface InitUploadRequest {
    filename: string
    size: number
    mime_type: string
    parent_id?: string
}

export interface InitUploadResponse {
    task_id: string
    chunk_size: number
    total_chunks: number
}

export async function initUpload(req: InitUploadRequest): Promise<InitUploadResponse> {
    const { data } = await client.post('/upload/init', req)
    return data
}

export async function uploadChunk(
    taskId: string,
    chunkIndex: number,
    chunk: Blob,
    onProgress?: (loaded: number) => void,
): Promise<void> {
    const formData = new FormData()
    formData.append('chunk_index', String(chunkIndex))
    formData.append('chunk', chunk)

    await client.post(`/upload/${taskId}/chunk`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (e) => {
            if (onProgress && e.loaded) {
                onProgress(e.loaded)
            }
        },
    })
}

export async function pauseUpload(taskId: string): Promise<void> {
    await client.put(`/upload/${taskId}/pause`)
}

export async function resumeUpload(taskId: string): Promise<void> {
    await client.put(`/upload/${taskId}/resume`)
}

export async function completeUpload(
    taskId: string,
    parentId?: string | null,
): Promise<{ file: import('@/types').FileItem; url: string; markdown: string }> {
    const { data } = await client.post(`/upload/${taskId}/complete`, {
        parent_id: parentId || undefined,
    })
    return data
}

export async function listTasks(): Promise<UploadTask[]> {
    const { data } = await client.get('/upload/tasks')
    return data.tasks || []
}
