import { useEffect, useState } from 'react'
import { Table, Button, Typography, Empty, Popconfirm, App, Pagination } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { FileItem } from '@/types'
import { listAllFiles, adminDeleteFile } from '@/api/admin'
import { formatBytes, formatDate } from '@/utils/format'

const { Title } = Typography

export default function AdminFiles() {
    const [files, setFiles] = useState<FileItem[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const pageSize = 50
    const { message } = App.useApp()

    const fetchFiles = async (p: number) => {
        setLoading(true)
        try {
            const data = await listAllFiles((p - 1) * pageSize, pageSize)
            setFiles(data.files)
            setTotal(data.total)
            setPage(p)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchFiles(1) }, [])

    const handleDelete = async (id: string) => {
        await adminDeleteFile(id)
        message.success('已删除')
        fetchFiles(page)
    }

    const columns = [
        {
            title: '文件名',
            dataIndex: 'name',
            key: 'name',
            ellipsis: true,
        },
        {
            title: '用户',
            dataIndex: 'user_id',
            key: 'user_id',
            width: 120,
            ellipsis: true,
            render: (id: string) => id.slice(0, 8) + '...',
        },
        {
            title: '类型',
            dataIndex: 'is_directory',
            key: 'type',
            width: 80,
            render: (isDir: boolean) => isDir ? '文件夹' : '文件',
        },
        {
            title: '大小',
            dataIndex: 'size',
            key: 'size',
            width: 100,
            render: (size: number, record: FileItem) => record.is_directory ? '-' : formatBytes(size),
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (date: string) => formatDate(date),
        },
        {
            title: '操作',
            key: 'actions',
            width: 80,
            render: (_: unknown, record: FileItem) => (
                <Popconfirm title="永久删除此文件？" onConfirm={() => handleDelete(record.id)} okText="删除" cancelText="取消">
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
            ),
        },
    ]

    return (
        <div className="fade-in">
            <Title level={4} style={{ marginBottom: 16 }}>全局文件管理</Title>
            <div style={{ background: 'var(--color-bg-container)', borderRadius: 'var(--border-radius)', border: '1px solid var(--color-border-secondary)' }}>
                <Table
                    dataSource={files}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    pagination={false}
                    locale={{ emptyText: <Empty description="暂无文件" /> }}
                />
            </div>
            {total > pageSize && (
                <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Pagination current={page} total={total} pageSize={pageSize} onChange={fetchFiles} showTotal={(t) => `共 ${t} 个文件`} />
                </div>
            )}
        </div>
    )
}
