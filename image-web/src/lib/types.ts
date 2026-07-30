export interface User {
  id: string
  username: string
  display_name: string
  email: string
  role: 'user' | 'admin'
  storage_quota: number
  storage_used: number
}

export interface FileItem {
  id: string
  user_id: string
  parent_id: string | null
  name: string
  is_directory: boolean
  mime_type?: string
  size: number
  public_url?: string
  moderation_status: 'normal' | 'banned'
  moderation_reason?: string
  deleted_at?: string | null
  created_at: string
  updated_at: string
}

export interface Share {
  id: string
  file_id: string
  code: string
  expires_at?: string | null
  view_count: number
  created_at: string
  file_name?: string
  has_password?: boolean
}

export interface ServerUploadTask {
  id: string
  filename: string
  total_size: number
  uploaded_size: number
  chunk_size: number
  total_chunks: number
  uploaded_chunks: number
  status: 'pending' | 'uploading' | 'paused' | 'completed' | 'failed'
}

export interface UploadTask {
  localId: string
  serverId?: string
  file: File
  parentId: string | null
  filename: string
  totalSize: number
  uploadedSize: number
  chunkSize: number
  totalChunks: number
  uploadedChunks: number
  status: 'pending' | 'uploading' | 'paused' | 'completed' | 'failed'
  progress: number
  speed: number
  error?: string
  url?: string
  markdown?: string
}

export type ViewName = 'files' | 'shares' | 'trash'
export type ViewMode = 'grid' | 'table'
