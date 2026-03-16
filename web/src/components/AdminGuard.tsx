import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'

/**
 * Route guard that requires admin role.
 * Redirects to dashboard if not admin.
 */
export default function AdminGuard() {
    const { isAdmin } = useAuthStore()

    if (!isAdmin) {
        return <Navigate to="/dashboard" replace />
    }

    return <Outlet />
}
