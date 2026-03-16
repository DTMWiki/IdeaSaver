// TypeScript types mapping backend Go models

export interface User {
    id: string
    username: string
    display_name: string
    email: string
    role: 'user' | 'admin'
    storage_quota: number
    storage_used: number
    created_at: string
    updated_at: string
}

export interface FileItem {
    id: string
    user_id: string
    parent_id: string | null
    name: string
    storage_key?: string
    is_directory: boolean
    mime_type?: string
    size: number
    public_url?: string
    thumbnail_key?: string
    deleted_at?: string | null
    created_at: string
    updated_at: string
}

export interface Share {
    id: string
    user_id: string
    file_id: string
    code: string
    expires_at?: string | null
    view_count: number
    created_at: string
    // file info (joined from server)
    file_name?: string
}

export interface UploadTask {
    id: string
    user_id: string
    filename: string
    total_size: number
    uploaded_size: number
    chunk_size: number
    total_chunks: number
    uploaded_chunks: number
    storage_key: string
    upload_id: string
    status: 'pending' | 'uploading' | 'paused' | 'completed' | 'failed'
    target_type: 'file' | 'video'
    created_at: string
    updated_at: string
}

export interface Video {
    id: string
    user_id: string
    title: string
    vid: string
    vcode: string
    status: number // 0=disabled 1=enabled
    play_url?: string
    size: number
    created_at: string
    updated_at: string
}

export interface AuditLog {
    id: number
    user_id: string
    action: string
    resource?: string
    resource_id?: string
    details?: Record<string, unknown>
    ip_address?: string
    user_agent?: string
    created_at: string
    username?: string
}

// Breadcrumb item for file navigation
export interface BreadcrumbItem {
    id: string | null // null = root
    name: string
}

// Upload task state (frontend-side tracking)
export interface UploadFileTask {
    id: string // local ID before server assigns one
    taskId?: string // server-assigned task ID
    file: File
    parentId: string | null
    filename: string
    totalSize: number
    uploadedSize: number
    chunkSize: number
    totalChunks: number
    uploadedChunks: number
    status: 'pending' | 'uploading' | 'paused' | 'completed' | 'failed'
    progress: number // 0-100
    speed: number // bytes per second
    url?: string
    markdown?: string
    error?: string
}
