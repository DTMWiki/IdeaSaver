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
    moderation_status: 'normal' | 'banned'
    moderation_reason?: string
    moderated_by?: string | null
    moderated_at?: string | null
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
    player_user_id?: string
    thumbnail_url?: string
    thumbnail_small_url?: string
    transcode_status: 'pending' | 'processing' | 'ready' | 'failed' | 'blocked'
    transcode_message?: string
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
    details?: unknown
    ip_address?: string
    user_agent?: string
    created_at: string
    username?: string
}

export interface FileAppeal {
    id: string
    file_id: string
    user_id: string
    status: 'pending' | 'approved' | 'deleted' | 'rejected'
    reason: string
    admin_comment?: string
    reviewed_by?: string | null
    reviewed_at?: string | null
    created_at: string
    updated_at: string
    file_name?: string
    username?: string
    reviewer_name?: string
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
    targetType: 'file' | 'video'
    filename: string
    totalSize: number
    uploadedSize: number
    chunkSize: number
    totalChunks: number
    uploadedChunks: number
    status: 'pending' | 'uploading' | 'paused' | 'completed' | 'failed'
    progress: number // 0-100
    speed: number // bytes per second
    phase?: 'uploading' | 'processing' | 'waiting_transcode'
    detail?: string
    url?: string
    markdown?: string
    videoId?: string
    vcode?: string
    error?: string
}
