import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

/**
 * Format bytes to human-readable size string.
 */
export function formatBytes(bytes: number, decimals = 1): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i]
}

/**
 * Format date string to readable format.
 */
export function formatDate(date: string): string {
    const parsed = dayjs(date)
    return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : '--'
}

/**
 * Format date to relative time (e.g. "3 小时前").
 */
export function formatRelativeTime(date: string): string {
    const parsed = dayjs(date)
    return parsed.isValid() ? parsed.fromNow() : '时间未知'
}

/**
 * Get file extension from filename.
 */
export function getFileExtension(filename: string): string {
    const idx = filename.lastIndexOf('.')
    return idx >= 0 ? filename.slice(idx + 1).toLowerCase() : ''
}

/**
 * Check if mime type is an image.
 */
export function isImage(mimeType?: string): boolean {
    return !!mimeType && mimeType.startsWith('image/')
}

/**
 * Check if mime type is audio.
 */
export function isAudio(mimeType?: string): boolean {
    return !!mimeType && mimeType.startsWith('audio/')
}

/**
 * Check if mime type is video.
 */
export function isVideo(mimeType?: string): boolean {
    return !!mimeType && mimeType.startsWith('video/')
}

/**
 * Check if mime type is text-based (for preview).
 */
export function isText(mimeType?: string): boolean {
    if (!mimeType) return false
    if (mimeType.startsWith('text/')) return true
    const textTypes = [
        'application/json',
        'application/javascript',
        'application/typescript',
        'application/xml',
        'application/yaml',
        'application/x-yaml',
        'application/toml',
    ]
    return textTypes.includes(mimeType)
}

/**
 * Copy text to clipboard.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text)
        return true
    } catch {
        // Fallback
        const el = document.createElement('textarea')
        el.value = text
        el.style.position = 'fixed'
        el.style.left = '-9999px'
        document.body.appendChild(el)
        el.select()
        document.execCommand('copy')
        document.body.removeChild(el)
        return true
    }
}

/**
 * Format bytes per second to speed string.
 */
export function formatSpeed(bytesPerSecond: number): string {
    if (bytesPerSecond <= 0) return '0 B/s'
    return formatBytes(bytesPerSecond) + '/s'
}
