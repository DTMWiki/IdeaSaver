import client from './client'
import type { Share, ShareAccessResult, ShareFileView } from '@/types'

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
): Promise<ShareAccessResult> {
    const headers: Record<string, string> = {}
    if (password) headers['X-Share-Password'] = password
    // Password via header only — never query string.
    const { data } = await client.get(`/shares/${encodeURIComponent(code)}`, { headers })
    return {
        file: data.file as ShareFileView,
        download_token: data.download_token as string,
        token_expires_in: Number(data.token_expires_in) || 0,
    }
}

/**
 * Build download URL using a short-lived token so the browser can stream
 * large files natively (no full-file JS blob). Token is not the share password.
 */
export function buildShareDownloadURL(
    code: string,
    token: string,
    options?: { inline?: boolean },
): string {
    const params = new URLSearchParams({ token })
    if (options?.inline) params.set('inline', '1')
    return `/api/shares/${encodeURIComponent(code)}/download?${params.toString()}`
}
