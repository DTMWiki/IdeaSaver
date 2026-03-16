import { create } from 'zustand'

interface ThemeState {
    isDark: boolean
    toggle: () => void
}

function getInitialTheme(): boolean {
    const saved = localStorage.getItem('theme')
    if (saved) return saved === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(isDark: boolean) {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
    localStorage.setItem('theme', isDark ? 'dark' : 'light')
}

// Apply theme on module load
const initialDark = getInitialTheme()
applyTheme(initialDark)

export const useThemeStore = create<ThemeState>((set, get) => ({
    isDark: initialDark,
    toggle: () => {
        const next = !get().isDark
        applyTheme(next)
        set({ isDark: next })
    },
}))
