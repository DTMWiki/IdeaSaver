import { lazy, Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Spin } from 'antd'
import AuthGuard from '@/components/AuthGuard'
import AdminGuard from '@/components/AdminGuard'

// Lazy-loaded pages
const Home = lazy(() => import('@/pages/Home'))
const LoginCallback = lazy(() => import('@/pages/LoginCallback'))
const AppLayout = lazy(() => import('@/layouts/AppLayout'))
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const VideoPage = lazy(() => import('@/pages/VideoPage'))
const TrashPage = lazy(() => import('@/pages/TrashPage'))
const SharesPage = lazy(() => import('@/pages/SharesPage'))
const HistoryPage = lazy(() => import('@/pages/HistoryPage'))
const ShareAccess = lazy(() => import('@/pages/ShareAccess'))
const AdminFiles = lazy(() => import('@/pages/admin/AdminFiles'))
const AdminVideos = lazy(() => import('@/pages/admin/AdminVideos'))
const AdminUsers = lazy(() => import('@/pages/admin/AdminUsers'))
const AdminLogs = lazy(() => import('@/pages/admin/AdminLogs'))
const AdminAppeals = lazy(() => import('@/pages/admin/AdminAppeals'))

function PageLoading() {
  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" />
    </div>
  )
}

export default function App() {
  return (
    <Suspense fallback={<PageLoading />}>
      <Routes>
        {/* Public routes */}
        <Route path="/" element={<Home />} />
        <Route path="/login/callback" element={<LoginCallback />} />
        <Route path="/share/:code" element={<ShareAccess />} />

        {/* Authenticated routes */}
        <Route element={<AuthGuard />}>
          <Route element={<AppLayout />}>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/dashboard/videos" element={<VideoPage />} />
            <Route path="/dashboard/trash" element={<TrashPage />} />
            <Route path="/dashboard/shares" element={<SharesPage />} />
            <Route path="/dashboard/history" element={<HistoryPage />} />

            {/* Admin routes */}
            <Route element={<AdminGuard />}>
              <Route path="/admin/files" element={<AdminFiles />} />
              <Route path="/admin/videos" element={<AdminVideos />} />
              <Route path="/admin/users" element={<AdminUsers />} />
              <Route path="/admin/logs" element={<AdminLogs />} />
              <Route path="/admin/appeals" element={<AdminAppeals />} />
            </Route>
          </Route>
        </Route>
      </Routes>
    </Suspense>
  )
}
