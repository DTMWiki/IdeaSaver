import client from './client'
import type { Video } from '@/types'

export interface VideoPlayInfo {
    ready: boolean
    play_url?: string
    vcode?: string
    player_user_id?: string
    message?: string
}

export async function uploadVideo(
    file: File,
    title?: string,
): Promise<Video> {
    const formData = new FormData()
    formData.append('file', file)
    if (title) formData.append('title', title)

    const { data } = await client.post('/videos/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 300000, // 5 min for large videos
    })
    return data.video
}

export async function listVideos(
    offset = 0,
    limit = 20,
): Promise<{ videos: Video[]; total: number }> {
    const { data } = await client.get('/videos/list', { params: { offset, limit } })
    return { videos: data.videos || [], total: data.total || 0 }
}

export async function setVideoStatus(id: string, status: number): Promise<void> {
    await client.put(`/videos/${id}/status`, { status })
}

export async function deleteVideo(id: string): Promise<void> {
    await client.delete(`/videos/${id}`)
}

export async function batchDeleteVideos(ids: string[]): Promise<void> {
    await client.delete('/videos/batch', { data: { ids } })
}

export async function getPlayURL(id: string): Promise<string> {
    const info = await getPlayInfo(id)
    if (!info.play_url) {
        throw new Error(info.message || '播放地址尚未就绪')
    }
    return info.play_url
}

export async function getPlayInfo(id: string): Promise<VideoPlayInfo> {
    const { data } = await client.get(`/videos/${id}/play-info`)
    return data
}
