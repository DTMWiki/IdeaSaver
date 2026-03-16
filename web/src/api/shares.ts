import client from './client'
import type { Share, FileItem } from '@/types'

export async function listShares(): Promise<Share[]> {
    const { data } = await client.get('/shares')
    return data.shares || []
}

export async function deleteShare(id: string): Promise<void> {
    await client.delete(`/shares/${id}`)
}

export async function accessShare(
    code: string,
    password?: string,
): Promise<FileItem> {
    const params: Record<string, string> = {}
    if (password) params.password = password
    const { data } = await client.get(`/shares/${code}`, { params })
    return data.file
}
