import { useEffect, useState } from 'react'
import { Modal, Spin, Typography, Image } from 'antd'
import type { FileItem } from '@/types'
import { getPreviewURL } from '@/api/files'
import { isImage, isAudio, isText } from '@/utils/format'

const { Text, Paragraph } = Typography

interface FilePreviewProps {
    file: FileItem | null
    onClose: () => void
}

export default function FilePreview({ file, onClose }: FilePreviewProps) {
    const [textContent, setTextContent] = useState<string | null>(null)
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        if (!file || file.is_directory) return
        setTextContent(null)

        if (isText(file.mime_type)) {
            setLoading(true)
            fetch(getPreviewURL(file.id))
                .then((r) => r.text())
                .then((text) => {
                    setTextContent(text)
                    setLoading(false)
                })
                .catch(() => {
                    setTextContent('无法加载文件内容')
                    setLoading(false)
                })
        }
    }, [file])

    if (!file) return null

    const renderContent = () => {
        if (loading) {
            return <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
        }

        if (isImage(file.mime_type)) {
            return (
                <div style={{ textAlign: 'center', padding: 16 }}>
                    <Image
                        src={getPreviewURL(file.id)}
                        alt={file.name}
                        style={{ maxHeight: '60vh', objectFit: 'contain' }}
                        fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mN88P/BfwAJhAPkGz0hkgAAAABJRU5ErkJggg=="
                    />
                </div>
            )
        }

        if (isAudio(file.mime_type)) {
            return (
                <div style={{ textAlign: 'center', padding: 40 }}>
                    <audio controls src={getPreviewURL(file.id)} style={{ width: '100%', maxWidth: 500 }}>
                        您的浏览器不支持音频播放
                    </audio>
                </div>
            )
        }

        if (isText(file.mime_type) && textContent !== null) {
            return (
                <div style={{ maxHeight: '60vh', overflow: 'auto' }}>
                    <pre
                        style={{
                            padding: 16,
                            background: 'var(--color-bg-spotlight)',
                            borderRadius: 'var(--border-radius-sm)',
                            fontSize: 13,
                            lineHeight: 1.6,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-all',
                            margin: 0,
                        }}
                    >
                        <code>{textContent}</code>
                    </pre>
                </div>
            )
        }

        return (
            <div style={{ textAlign: 'center', padding: 40 }}>
                <Paragraph type="secondary">此文件类型暂不支持预览</Paragraph>
                <Text type="secondary" style={{ fontSize: 12 }}>
                    MIME: {file.mime_type || '未知'}
                </Text>
            </div>
        )
    }

    return (
        <Modal
            title={file.name}
            open={!!file}
            onCancel={onClose}
            footer={null}
            width={720}
            destroyOnClose
        >
            {renderContent()}
        </Modal>
    )
}
