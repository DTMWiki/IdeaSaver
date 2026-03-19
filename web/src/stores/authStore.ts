import { create } from 'zustand'
import type { User } from '@/types'
import * as authApi from '@/api/auth'

interface AuthState {
    token: string | null
    user: User | null
    loading: boolean
    isLoggedIn: boolean
    isAdmin: boolean

    login: () => Promise<void>
    logout: () => void
    handleCallback: (code: string) => Promise<void>
    fetchMe: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
    token: localStorage.getItem('token'),
    user: null,
    loading: false,
    isLoggedIn: !!localStorage.getItem('token'),
    isAdmin: false,

    login: async () => {
        const url = await authApi.getLoginURL()
        window.location.href = url
    },

    logout: () => {
        localStorage.removeItem('token')
        set({ token: null, user: null, isLoggedIn: false, isAdmin: false })
        window.location.href = '/'
    },

    handleCallback: async (code: string) => {
        set({ loading: true })
        try {
            const { token, user } = await authApi.exchangeCode(code)
            localStorage.setItem('token', token)
            set({
                token,
                user,
                isLoggedIn: true,
                isAdmin: user.role === 'admin',
                loading: false,
            })
        } catch {
            set({ loading: false })
            throw new Error('登录失败')
        }
    },

    fetchMe: async () => {
        if (!get().token) return
        set({ loading: true })
        try {
            const user = await authApi.getMe()
            set({ user, isLoggedIn: true, isAdmin: user.role === 'admin', loading: false })
        } catch {
            // Token might be invalid
            localStorage.removeItem('token')
            set({ token: null, user: null, isLoggedIn: false, isAdmin: false, loading: false })
        }
    },
}))
