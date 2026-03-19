import { useEffect, useState } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { Spin } from 'antd'
import { useAuthStore } from '@/stores/authStore'

/**
 * Route guard that requires authentication.
 * Redirects to home page if not logged in.
 */
export default function AuthGuard() {
    const { isLoggedIn, user, fetchMe } = useAuthStore()
    const [checking, setChecking] = useState(true)

    useEffect(() => {
        if (isLoggedIn && !user) {
            fetchMe().finally(() => setChecking(false))
        } else {
            setChecking(false)
        }
    }, [isLoggedIn, user, fetchMe])

    if (checking) {
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
