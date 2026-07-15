import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Space, Button, Typography, Tooltip } from 'antd'
import {
    FileOutlined,
    VideoCameraOutlined,
    DeleteOutlined,
    ShareAltOutlined,
    HistoryOutlined,
    DashboardOutlined,
    UserOutlined,
    AuditOutlined,
    TeamOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    SunOutlined,
    MoonOutlined,
    FlagOutlined,
} from '@ant-design/icons'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faLightbulb } from '@fortawesome/free-solid-svg-icons'
import { useAuthStore } from '@/stores/authStore'
import { useThemeStore } from '@/stores/themeStore'
import QuotaIndicator from '@/components/QuotaIndicator'
import SSEProvider from '@/components/SSEProvider'
import UploadPanel from '@/components/UploadPanel'
import './AppLayout.css'

const { Header, Sider, Content } = Layout
const { Text } = Typography

export default function AppLayout() {
    const [collapsed, setCollapsed] = useState(false)
    const navigate = useNavigate()
    const location = useLocation()
    const { user, isAdmin, logout } = useAuthStore()
    const { isDark, toggle: toggleTheme } = useThemeStore()

    const userMenuItems = [
        {
            key: 'files',
            icon: <FileOutlined />,
            label: '文件管理',
            onClick: () => navigate('/dashboard'),
        },
        {
            key: 'videos',
            icon: <VideoCameraOutlined />,
            label: '视频管理',
            onClick: () => navigate('/dashboard/videos'),
        },
        {
            key: 'trash',
            icon: <DeleteOutlined />,
            label: '回收站',
            onClick: () => navigate('/dashboard/trash'),
        },
        {
            key: 'shares',
            icon: <ShareAltOutlined />,
            label: '我的分享',
            onClick: () => navigate('/dashboard/shares'),
        },
        {
            key: 'history',
            icon: <HistoryOutlined />,
            label: '操作历史',
            onClick: () => navigate('/dashboard/history'),
        },
    ]

    const adminMenuItems = isAdmin
        ? [
            { type: 'divider' as const },
            {
                key: 'admin-header',
                type: 'group' as const,
                label: '管理员',
                children: [
                    {
                        key: 'admin-files',
                        icon: <DashboardOutlined />,
                        label: '全局文件',
                        onClick: () => navigate('/admin/files'),
                    },
                    {
                        key: 'admin-videos',
                        icon: <VideoCameraOutlined />,
                        label: '全局视频',
                        onClick: () => navigate('/admin/videos'),
                    },
                    {
                        key: 'admin-users',
                        icon: <TeamOutlined />,
                        label: '用户管理',
                        onClick: () => navigate('/admin/users'),
                    },
                    {
                        key: 'admin-logs',
                        icon: <AuditOutlined />,
                        label: '审计日志',
                        onClick: () => navigate('/admin/logs'),
                    },
                    {
                        key: 'admin-appeals',
                        icon: <FlagOutlined />,
                        label: '申诉工单',
                        onClick: () => navigate('/admin/appeals'),
                    },
                ],
            },
        ]
        : []

    // Determine selected menu key from pathname
    const getSelectedKey = () => {
        const path = location.pathname
        if (path === '/dashboard') return 'files'
        if (path.includes('/videos') && !path.includes('/admin')) return 'videos'
        if (path.includes('/trash')) return 'trash'
        if (path.includes('/shares')) return 'shares'
        if (path.includes('/history')) return 'history'
        if (path.includes('/admin/files')) return 'admin-files'
        if (path.includes('/admin/videos')) return 'admin-videos'
        if (path.includes('/admin/users')) return 'admin-users'
        if (path.includes('/admin/logs')) return 'admin-logs'
        if (path.includes('/admin/appeals')) return 'admin-appeals'
        return 'files'
    }

    const avatarDropdownItems = [
        {
            key: 'user-info',
            label: (
                <div style={{ padding: '4px 0' }}>
                    <div style={{ fontWeight: 600 }}>{user?.display_name || user?.username}</div>
                    <div style={{ fontSize: 12, opacity: 0.6 }}>{user?.email}</div>
                </div>
            ),
            disabled: true,
        },
        { type: 'divider' as const },
        {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: '退出登录',
            danger: true,
            onClick: logout,
        },
    ]

    return (
        <SSEProvider>
            <Layout className="app-layout">
                <Sider
                    trigger={null}
                    collapsible
                    collapsed={collapsed}
                    width={240}
                    collapsedWidth={64}
                    className="app-sider"
                    breakpoint="lg"
                    onBreakpoint={(broken) => setCollapsed(broken)}
                >
                    <div className="sider-logo" onClick={() => navigate('/dashboard')}>
                        <span className="logo-mark" aria-hidden="true">
                            <FontAwesomeIcon icon={faLightbulb} />
                        </span>
                        {!collapsed && <span className="logo-text">IdeaSaver</span>}
                    </div>

                    <Menu
                        mode="inline"
                        selectedKeys={[getSelectedKey()]}
                        items={[...userMenuItems, ...adminMenuItems]}
                        className="sider-menu"
                    />

                    {!collapsed && (
                        <div className="sider-footer">
                            <QuotaIndicator />
                        </div>
                    )}
                </Sider>

                <Layout>
                    <Header className="app-header">
                        <Button
                            type="text"
                            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                            onClick={() => setCollapsed(!collapsed)}
                            className="collapse-btn"
                        />

                        <div className="header-spacer" />

                        <Space size={12} align="center">
                            {collapsed && <QuotaIndicator compact />}

                            <Tooltip title={isDark ? '切换亮色模式' : '切换暗色模式'}>
                                <Button
                                    type="text"
                                    icon={isDark ? <SunOutlined /> : <MoonOutlined />}
                                    onClick={toggleTheme}
                                    className="theme-btn"
                                />
                            </Tooltip>

                            <Dropdown menu={{ items: avatarDropdownItems }} trigger={['click']} placement="bottomRight">
                                <Space className="user-avatar-trigger" style={{ cursor: 'pointer' }}>
                                    <Avatar size={32} icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }}>
                                        {user?.display_name?.[0] || user?.username?.[0] || 'U'}
                                    </Avatar>
                                    {!collapsed && (
                                        <Text className="username-text" ellipsis>
                                            {user?.display_name || user?.username}
                                        </Text>
                                    )}
                                </Space>
                            </Dropdown>
                        </Space>
                    </Header>

                    <Content className="app-content">
                        <Outlet />
                    </Content>
                </Layout>

                <UploadPanel />
            </Layout>
        </SSEProvider>
    )
}
