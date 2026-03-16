import client from './client'
import type { FileItem, Video, AuditLog, User } from '@/types'

export async function listAllFiles(
    offset = 0,
    limit = 50,
): Promise<{ files: FileItem[]; total: number }> {
    const { data } = await client.get('/admin/files', { params: { offset, limit } })
    return { files: data.files || [], total: data.total || 0 }
}

export async function adminDeleteFile(id: string): Promise<void> {
    await client.delete(`/admin/files/${id}`)
}

export async function listAllVideos(
    offset = 0,
    limit = 50,
): Promise<{ videos: Video[]; total: number }> {
    const { data } = await client.get('/admin/videos', { params: { offset, limit } })
    return { videos: data.videos || [], total: data.total || 0 }
}

export async function adminDeleteVideo(id: string): Promise<void> {
    await client.delete(`/admin/videos/${id}`)
}

export async function listLogs(
    action?: string,
    offset = 0,
    limit = 50,
): Promise<{ logs: AuditLog[]; total: number }> {
    const params: Record<string, string | number> = { offset, limit }
    if (action) params.action = action
    const { data } = await client.get('/admin/logs', { params })
    return { logs: data.logs || [], total: data.total || 0 }
}

export async function listUsers(
    offset = 0,
    limit = 50,
): Promise<{ users: User[]; total: number }> {
    const { data } = await client.get('/admin/users', { params: { offset, limit } })
    return { users: data.users || [], total: data.total || 0 }
}

export async function updateUserQuota(id: string, quota: number): Promise<void> {
    await client.put(`/admin/users/${id}/quota`, { quota })
}

export async function cleanupTrash(): Promise<number> {
    const { data } = await client.delete('/admin/trash/cleanup')
    return data.cleaned
}
