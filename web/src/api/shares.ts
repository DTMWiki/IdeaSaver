import client from './client'
import type { Share, ShareFileView } from '@/types'

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
): Promise<ShareFileView> {
    const params: Record<string, string> = {}
    if (password) params.password = password
    const { data } = await client.get(`/shares/${code}`, { params })
    return data.file
}

/** Authenticated download URL that re-checks share password/expiry server-side. */
export function buildShareDownloadURL(code: string, password?: string): string {
    const params = new URLSearchParams()
    if (password) params.set('password', password)
    const query = params.toString()
    return `/api/shares/${encodeURIComponent(code)}/download${query ? `?${query}` : ''}`
}
