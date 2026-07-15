import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Spin, Result, Button } from 'antd'
import { useAuthStore } from '@/stores/authStore'

export default function LoginCallback() {
    const [searchParams] = useSearchParams()
    const navigate = useNavigate()
    const { handleCallback, isLoggedIn } = useAuthStore()
    const [error, setError] = useState<string | null>(null)
    const [submitting, setSubmitting] = useState(false)

    const code = searchParams.get('code')
    const state = searchParams.get('state')

    useEffect(() => {
        if (!code || submitting || isLoggedIn || error) return

        setSubmitting(true)
        handleCallback(code, state)
            .then(() => {
                navigate('/dashboard', { replace: true })
            })
            .catch((err: unknown) => {
                const msg =
                    (err as { message?: string })?.message ||
                    '登录失败，请重试'
                setError(msg)
                setSubmitting(false)
            })
    }, [code, state, handleCallback, navigate, submitting, isLoggedIn, error])

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

    if (error) {
        return (
            <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Result
                    status="error"
                    title="登录失败"
                    subTitle={error}
                    extra={
                        <Button type="primary" onClick={() => navigate('/')}>
                            返回首页重新登录
                        </Button>
                    }
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
