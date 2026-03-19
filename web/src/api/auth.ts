import client from './client'
import type { User } from '@/types'

export async function getLoginURL(): Promise<string> {
    const { data } = await client.get('/auth/login')
    return data.url
}

export async function exchangeCode(code: string): Promise<{ token: string; user: User }> {
    const { data } = await client.get('/auth/callback', { params: { code } })
    return data
}

export async function getMe(): Promise<User> {
    const { data } = await client.get('/user/me')
    return data.user
}

export async function getQuota(): Promise<{ quota: number; used: number }> {
    const { data } = await client.get('/user/quota')
    return data
}

export async function getHistory(offset = 0, limit = 50) {
    const { data } = await client.get('/user/history', { params: { offset, limit } })
    return data as { logs: import('@/types').AuditLog[]; total: number }
}
