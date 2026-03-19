import { useEffect } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuthStore } from '@/stores/authStore'

/**
 * Route guard that requires authentication.
 * Redirects to home page if not logged in.
 */
export default function AuthGuard() {
    const { isLoggedIn, user, loading, fetchMe } = useAuthStore()

    useEffect(() => {
        if (isLoggedIn && !user && !loading) {
            void fetchMe()
        }
    }, [isLoggedIn, user, loading, fetchMe])

    if (isLoggedIn && (!user || loading)) {
        return (
            <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Spin size="large" tip="加载中..." />
            </div>
        )
    }

    if (!isLoggedIn) {
        return <Navigate to="/" replace />
    }

    return <Outlet />
}
