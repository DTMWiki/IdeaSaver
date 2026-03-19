import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Spin, Result, Button } from 'antd'
import { useAuthStore } from '@/stores/authStore'

export default function LoginCallback() {
    const [searchParams] = useSearchParams()
    const navigate = useNavigate()
    const { handleCallback, isLoggedIn } = useAuthStore()

    useEffect(() => {
        const code = searchParams.get('code')
        if (!code) return

        handleCallback(code)
            .then(() => {
                navigate('/dashboard', { replace: true })
            })
            .catch(() => {
                // Error is handled in state
            })
    }, [searchParams, handleCallback, navigate])

    const code = searchParams.get('code')

    if (!code) {
        return (
            <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Result
                    status="error"
                    title="登录失败"
                    subTitle="缺少授权码参数"
                    extra={<Button type="primary" onClick={() => navigate('/')}>返回首页</Button>}
                />
            </div>
        )
    }

    if (isLoggedIn) {
        return (
            <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Result
                    status="success"
                    title="登录成功"
                    subTitle="正在跳转..."
                />
            </div>
        )
    }

    return (
        <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Spin size="large" tip="正在登录..." />
        </div>
    )
}
