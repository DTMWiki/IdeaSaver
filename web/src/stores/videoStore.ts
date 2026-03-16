import { create } from 'zustand'
import type { Video } from '@/types'
import * as videosApi from '@/api/videos'

interface VideoState {
    videos: Video[]
    total: number
    loading: boolean
    page: number
    pageSize: number

    fetchVideos: (page?: number) => Promise<void>
    uploadVideo: (file: File, title?: string) => Promise<void>
    toggleStatus: (id: string, currentStatus: number) => Promise<void>
    deleteVideo: (id: string) => Promise<void>
    batchDelete: (ids: string[]) => Promise<void>
    setPage: (page: number) => void
}

export const useVideoStore = create<VideoState>((set, get) => ({
    videos: [],
    total: 0,
    loading: false,
    page: 1,
    pageSize: 20,

    fetchVideos: async (page?: number) => {
        const p = page ?? get().page
        const { pageSize } = get()
        set({ loading: true, page: p })
        try {
            const { videos, total } = await videosApi.listVideos((p - 1) * pageSize, pageSize)
            set({ videos, total, loading: false })
        } catch {
            set({ loading: false })
        }
    },

    uploadVideo: async (file: File, title?: string) => {
        set({ loading: true })
        try {
            await videosApi.uploadVideo(file, title)
            await get().fetchVideos(1)
        } finally {
            set({ loading: false })
        }
    },

    toggleStatus: async (id: string, currentStatus: number) => {
        const newStatus = currentStatus === 1 ? 0 : 1
        await videosApi.setVideoStatus(id, newStatus)
        set((state) => ({
            videos: state.videos.map((v) => (v.id === id ? { ...v, status: newStatus } : v)),
        }))
    },

    deleteVideo: async (id: string) => {
        await videosApi.deleteVideo(id)
        await get().fetchVideos()
    },

    batchDelete: async (ids: string[]) => {
        await videosApi.batchDeleteVideos(ids)
        await get().fetchVideos()
    },

    setPage: (page: number) => {
        get().fetchVideos(page)
    },
}))
