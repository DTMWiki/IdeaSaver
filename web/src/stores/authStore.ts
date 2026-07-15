import { create } from 'zustand'
import type { User } from '@/types'
import * as authApi from '@/api/auth'

interface AuthState {
    user: User | null
    loading: boolean
    bootstrapped: boolean
    isLoggedIn: boolean
    isAdmin: boolean

    login: () => Promise<void>
    logout: () => void
    handleCallback: (code: string, state?: string | null) => Promise<void>
    bootstrap: () => Promise<void>
    fetchMe: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    loading: false,
    bootstrapped: false,
    isLoggedIn: false,
    isAdmin: false,

    login: async () => {
        const url = await authApi.getLoginURL()
        window.location.href = url
    },

    logout: () => {
        void authApi.logoutSession().finally(() => {
            set({ user: null, isLoggedIn: false, isAdmin: false, bootstrapped: true })
            window.location.href = '/'
        })
    },

    handleCallback: async (code: string, state?: string | null) => {
        set({ loading: true })
        try {
            const { user } = await authApi.exchangeCode(code, state)
            set({
                user,
                isLoggedIn: true,
                isAdmin: user.role === 'admin',
                loading: false,
                bootstrapped: true,
            })
        } catch (err: unknown) {
            set({ loading: false, bootstrapped: true, isLoggedIn: false, user: null, isAdmin: false })
            const msg =
                (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
                '登录失败'
            throw new Error(msg)
        }
    },

    bootstrap: async () => {
        if (get().bootstrapped && !get().loading) {
            // Still allow refresh of session
        }
        set({ loading: true })
        try {
            const user = await authApi.getMe()
            set({
                user,
                isLoggedIn: true,
                isAdmin: user.role === 'admin',
                loading: false,
                bootstrapped: true,
            })
        } catch {
            set({
                user: null,
                isLoggedIn: false,
                isAdmin: false,
                loading: false,
                bootstrapped: true,
            })
        }
    },

    fetchMe: async () => {
        set({ loading: true })
        try {
            const user = await authApi.getMe()
            set({
                user,
                isLoggedIn: true,
                isAdmin: user.role === 'admin',
                loading: false,
                bootstrapped: true,
            })
        } catch {
            set({
                user: null,
                isLoggedIn: false,
                isAdmin: false,
                loading: false,
                bootstrapped: true,
            })
        }
    },
}))
