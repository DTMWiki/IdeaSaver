import { useState } from 'react'
import { List, Progress, Button, Space, Typography, Tooltip, Badge } from 'antd'
import {
    PauseCircleOutlined,
    PlayCircleOutlined,
    CloseOutlined,
    CopyOutlined,
    CheckCircleFilled,
    ExclamationCircleFilled,
    UpOutlined,
    DownOutlined,
    DeleteOutlined,
    CloudUploadOutlined,
    VideoCameraOutlined,
    FileOutlined,
} from '@ant-design/icons'
import { App } from 'antd'
import { useUploadStore } from '@/stores/uploadStore'
import { formatBytes, formatSpeed, copyToClipboard } from '@/utils/format'
import './UploadPanel.css'

const { Text } = Typography

export default function UploadPanel() {
    const { tasks, panelVisible, setPanelVisible, pauseTask, resumeTask, removeTask, clearCompleted } =
        useUploadStore()
    const [collapsed, setCollapsed] = useState(false)
    const { message } = App.useApp()

    if (!panelVisible || tasks.length === 0) return null

    const activeCount = tasks.filter((t) => t.status === 'uploading' || t.status === 'pending').length
    const completedCount = tasks.filter((t) => t.status === 'completed').length

    const handleCopy = async (text: string, label: string) => {
        await copyToClipboard(text)
        message.success(`${label} 已复制`)
    }

    return (
        <div className={`upload-panel ${collapsed ? 'collapsed' : ''}`}>
            <div className="upload-panel-header" onClick={() => setCollapsed(!collapsed)}>
                <Space>
                    <CloudUploadOutlined />
                    <Text strong>上传队列</Text>
                    {activeCount > 0 && (
                        <Badge count={activeCount} size="small" style={{ backgroundColor: '#1677ff' }} />
                    )}
                    {completedCount > 0 && (
                        <Badge count={`✓${completedCount}`} size="small" style={{ backgroundColor: '#52c41a' }} />
                    )}
                </Space>
                <Space>
                    {completedCount > 0 && (
                        <Button type="text" size="small" icon={<DeleteOutlined />} onClick={(e) => { e.stopPropagation(); clearCompleted() }}>
                            清理完成
                        </Button>
                    )}
                    <Button
                        type="text"
                        size="small"
                        icon={collapsed ? <UpOutlined /> : <DownOutlined />}
                    />
                    <Button
                        type="text"
                        size="small"
                        icon={<CloseOutlined />}
                        onClick={(e) => { e.stopPropagation(); setPanelVisible(false) }}
                    />
                </Space>
            </div>

            {!collapsed && (
                <div className="upload-panel-body">
                    <List
                        size="small"
                        dataSource={tasks}
                        renderItem={(task) => (
                            <List.Item className="upload-task-item">
                                <div className="upload-task-content">
                                    <div className="upload-task-row">
                                        <Space size={6} style={{ minWidth: 0 }}>
                                            {task.targetType === 'video' ? <VideoCameraOutlined /> : <FileOutlined />}
                                            <Text className="upload-filename" ellipsis={{ tooltip: task.filename }}>
                                                {task.filename}
                                            </Text>
                                        </Space>
                                        <Space size={4}>
                                            {task.status === 'uploading' && (
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    {formatSpeed(task.speed)}
                                                </Text>
                                            )}
                                            {task.status === 'uploading' && task.targetType === 'file' && (
                                                <Tooltip title="暂停">
                                                    <Button type="text" size="small" icon={<PauseCircleOutlined />} onClick={() => pauseTask(task.id)} />
                                                </Tooltip>
                                            )}
                                            {task.status === 'paused' && task.targetType === 'file' && (
                                                <Tooltip title="继续">
                                                    <Button type="text" size="small" icon={<PlayCircleOutlined />} onClick={() => resumeTask(task.id)} />
                                                </Tooltip>
                                            )}
                                            {task.status === 'completed' && (
                                                <>
                                                    <CheckCircleFilled style={{ color: '#52c41a' }} />
                                                    {task.url && (
                                                        <Tooltip title="复制直链">
                                                            <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => handleCopy(task.url!, '直链')} />
                                                        </Tooltip>
                                                    )}
                                                    {task.markdown && (
                                                        <Tooltip title="复制 Markdown">
                                                            <Button type="text" size="small" onClick={() => handleCopy(task.markdown!, 'Markdown')}>
                                                                MD
                                                            </Button>
                                                        </Tooltip>
                                                    )}
                                                </>
                                            )}
                                            {task.status === 'failed' && (
                                                <Tooltip title={task.error || '上传失败'}>
                                                    <ExclamationCircleFilled style={{ color: '#ff4d4f' }} />
                                                </Tooltip>
                                            )}
                                            <Tooltip title="移除">
                                                <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => removeTask(task.id)} />
                                            </Tooltip>
                                        </Space>
                                    </div>
                                    {(task.status === 'uploading' || task.status === 'paused' || task.phase === 'processing' || task.phase === 'waiting_transcode') && (
                                        <Progress
                                            percent={task.progress}
                                            size="small"
                                            status={task.status === 'paused' ? 'exception' : 'active'}
                                            format={(p) => `${p}% · ${formatBytes(task.uploadedSize)} / ${formatBytes(task.totalSize)}`}
                                        />
                                    )}
                                    {(task.detail || task.vcode) && (
                                        <Space direction="vertical" size={2} style={{ width: '100%', marginTop: 4 }}>
                                            {task.detail && (
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    {task.detail}
                                                </Text>
                                            )}
                                            {task.vcode && (
                                                <Text type="secondary" style={{ fontSize: 12 }}>
                                                    播放码: {task.vcode}
                                                </Text>
                                            )}
                                        </Space>
                                    )}
                                </div>
                            </List.Item>
                        )}
                    />
                </div>
            )}
        </div>
    )
}
