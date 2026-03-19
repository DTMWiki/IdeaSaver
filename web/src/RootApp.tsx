import { BrowserRouter } from 'react-router-dom'
import { ConfigProvider, theme as antdTheme, App as AntApp } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useThemeStore } from '@/stores/themeStore'
import App from './App'

export default function RootApp() {
    const isDark = useThemeStore((s) => s.isDark)

    return (
        <ConfigProvider
            locale={zhCN}
            theme={{
                algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
                token: {
                    colorPrimary: '#1677ff',
                    borderRadius: 8,
                    fontFamily: '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans CJK SC", "Source Han Sans SC", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
                },
            }}
        >
            <AntApp>
                <BrowserRouter>
                    <App />
                </BrowserRouter>
            </AntApp>
        </ConfigProvider>
    )
}
