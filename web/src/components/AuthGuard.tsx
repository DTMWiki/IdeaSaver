import { useEffect } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuthStore } from '@/stores/authStore'

/**
 * Route guard that requires authentication via HttpOnly session cookie.
 */
export default function AuthGuard() {
    const { isLoggedIn, user, loading, bootstrapped, bootstrap } = useAuthStore()

    useEffect(() => {
        if (!bootstrapped) {
            void bootstrap()
        } else if (isLoggedIn && !user && !loading) {
            void bootstrap()
        }
    }, [bootstrapped, isLoggedIn, user, loading, bootstrap])

    if (!bootstrapped || loading) {
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
