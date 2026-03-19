import { useState } from 'react'
import { VideoCameraOutlined } from '@ant-design/icons'

interface VideoThumbnailProps {
    src?: string
    alt: string
    width?: number | string
    height?: number | string
    borderRadius?: number
    iconSize?: number
}

export default function VideoThumbnail({
    src,
    alt,
    width = '100%',
    height = 180,
    borderRadius = 12,
    iconSize = 28,
}: VideoThumbnailProps) {
    const normalizedSrc = normalizeThumbnailURL(src)
    const [loadedSrc, setLoadedSrc] = useState('')
    const [failedSrc, setFailedSrc] = useState('')
    const loaded = normalizedSrc !== '' && loadedSrc === normalizedSrc
    const failed = !normalizedSrc || failedSrc === normalizedSrc

    return (
        <div
            style={{
                position: 'relative',
                width,
                height,
                overflow: 'hidden',
                borderRadius,
                background: 'linear-gradient(135deg, rgba(22,119,255,0.10), rgba(235,47,150,0.10))',
                border: '1px solid var(--color-border-secondary)',
            }}
        >
            <div
                style={{
                    position: 'absolute',
                    inset: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: failed ? 'rgba(0,0,0,0.35)' : 'rgba(22,119,255,0.55)',
                    background: failed ? 'var(--color-bg-spotlight)' : 'transparent',
                }}
            >
                <VideoCameraOutlined style={{ fontSize: iconSize }} />
            </div>
            {normalizedSrc && !failed && (
                <img
                    src={normalizedSrc}
                    alt={alt}
                    loading="lazy"
                    onLoad={() => setLoadedSrc(normalizedSrc)}
                    onError={() => {
                        setFailedSrc(normalizedSrc)
                        setLoadedSrc('')
                    }}
                    style={{
                        position: 'absolute',
                        inset: 0,
                        width: '100%',
                        height: '100%',
                        objectFit: 'cover',
                        opacity: loaded ? 1 : 0,
                        transition: 'opacity 0.2s ease',
                    }}
                />
            )}
        </div>
    )
}

function normalizeThumbnailURL(src?: string) {
    const value = src?.trim()
    if (!value) return ''
    if (value.startsWith('//')) {
        return `https:${value}`
    }
    return value
}
