import type { FileItem, ServerUploadTask, Share, User } from './types'

const rawBase = (import.meta.env.VITE_API_BASE || '/api').replace(/\/$/, '')
export const apiBase = rawBase

export class ApiError extends Error {
  constructor(message: string, readonly status: number, readonly code?: string) {
    super(message)
    this.name = 'ApiError'
  }
}

export function apiUrl(path: string): string {
  return `${rawBase}${path.startsWith('/') ? path : `/${path}`}`
}

async function authorizedFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const headers = new Headers(options.headers)
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  return fetch(apiUrl(path), { ...options, headers, credentials: 'include' })
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await authorizedFetch(path, options)
  if (!response.ok) {
    const payload = await response.json().catch(() => ({})) as { error?: string; message?: string; code?: string }
    throw new ApiError(payload.error || payload.message || `请求失败 (${response.status})`, response.status, payload.code)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const authApi = {
  async me(): Promise<User> {
    return (await request<{ user: User }>('/user/me')).user
  },
  async quota(): Promise<{ quota: number; used: number }> {
    return request('/user/quota')
  },
  async login(): Promise<string> {
    return (await request<{ url: string }>('/auth/login')).url
  },
  async callback(code: string, state?: string | null): Promise<User> {
    const params = new URLSearchParams({ code })
    if (state) params.set('state', state)
    return (await request<{ user: User }>(`/auth/callback?${params}`)).user
  },
  async logout(): Promise<void> {
    await request('/auth/logout', { method: 'POST' })
  },
}

export const filesApi = {
  async list(parentId?: string | null): Promise<FileItem[]> {
    const query = parentId ? `?parent_id=${encodeURIComponent(parentId)}` : ''
    return (await request<{ files?: FileItem[] }>(`/files/list${query}`)).files || []
  },
  async mkdir(name: string, parentId?: string | null): Promise<FileItem> {
    return (await request<{ directory: FileItem }>('/files/mkdir', {
      method: 'POST', body: JSON.stringify({ name, parent_id: parentId || undefined }),
    })).directory
  },
  async rename(id: string, name: string): Promise<void> {
    await request(`/files/${id}/rename`, { method: 'PUT', body: JSON.stringify({ name }) })
  },
  async move(id: string, parentId: string | null): Promise<void> {
    await request(`/files/${id}/move`, { method: 'PUT', body: JSON.stringify({ parent_id: parentId }) })
  },
  async copy(id: string, parentId?: string | null): Promise<FileItem> {
    return (await request<{ file: FileItem }>('/files/copy', {
      method: 'POST', body: JSON.stringify({ file_id: id, parent_id: parentId || undefined }),
    })).file
  },
  async softDelete(id: string): Promise<void> {
    await request(`/files/${id}`, { method: 'DELETE' })
  },
  async batchDelete(ids: string[]): Promise<{ ok: boolean; deleted: number; failed: string[] }> {
    return request('/files/batch', { method: 'DELETE', body: JSON.stringify({ ids }) })
  },
  async trash(): Promise<FileItem[]> {
    return (await request<{ files?: FileItem[] }>('/files/trash')).files || []
  },
  async restore(id: string): Promise<void> {
    await request(`/files/${id}/restore`, { method: 'POST' })
  },
  async permanentDelete(id: string): Promise<void> {
    await request(`/files/${id}/permanent`, { method: 'DELETE' })
  },
  async link(id: string): Promise<{ url: string; markdown: string }> {
    return request(`/files/${id}/url`)
  },
  async preview(id: string): Promise<Blob> {
    const response = await authorizedFetch(`/files/${id}/preview`)
    if (!response.ok) throw new Error(`预览加载失败 (${response.status})`)
    return response.blob()
  },
  async thumbnail(id: string): Promise<Blob> {
    const response = await authorizedFetch(`/files/${id}/preview?thumb=1`)
    if (!response.ok) throw new Error(`缩略图加载失败 (${response.status})`)
    return response.blob()
  },
  async createShare(id: string, password?: string, expiresIn?: number): Promise<Share> {
    return (await request<{ share: Share }>(`/files/${id}/share`, {
      method: 'POST', body: JSON.stringify({ password: password || undefined, expires_in: expiresIn || undefined }),
    })).share
  },
}

export const sharesApi = {
  async list(): Promise<Share[]> {
    return (await request<{ shares?: Share[] }>('/shares')).shares || []
  },
  async remove(id: string): Promise<void> {
    await request(`/shares/${id}`, { method: 'DELETE' })
  },
  async access(code: string, password?: string): Promise<{ file: { name: string; size: number; mime_type?: string; is_image: boolean }; download_token: string; token_expires_in: number }> {
    return request(`/shares/${encodeURIComponent(code)}`, {
      headers: password ? { 'X-Share-Password': password } : undefined,
    })
  },
  downloadUrl(code: string, token: string, inline = false): string {
    const params = new URLSearchParams({ token })
    if (inline) params.set('inline', '1')
    return apiUrl(`/shares/${encodeURIComponent(code)}/download?${params}`)
  },
}

export const uploadApi = {
  async init(file: File, parentId: string | null): Promise<{ task_id: string; chunk_size: number; total_chunks: number }> {
    return request('/upload/init', {
      method: 'POST',
      body: JSON.stringify({
        filename: file.name,
        size: file.size,
        mime_type: file.type || 'application/octet-stream',
        parent_id: parentId || undefined,
      }),
    })
  },
  async pause(taskId: string): Promise<void> {
    await request(`/upload/${taskId}/pause`, { method: 'PUT' })
  },
  async resume(taskId: string): Promise<void> {
    await request(`/upload/${taskId}/resume`, { method: 'PUT' })
  },
  async complete(taskId: string, parentId: string | null): Promise<{ file: FileItem; url: string; markdown: string }> {
    return request(`/upload/${taskId}/complete`, {
      method: 'POST', body: JSON.stringify({ parent_id: parentId || undefined }),
    })
  },
  async tasks(): Promise<ServerUploadTask[]> {
    return (await request<{ tasks?: ServerUploadTask[] }>('/upload/tasks')).tasks || []
  },
  uploadChunk(taskId: string, index: number, chunk: Blob, onProgress: (loaded: number) => void): { promise: Promise<void>; abort: () => void } {
    const xhr = new XMLHttpRequest()
    const data = new FormData()
    data.append('chunk_index', String(index))
    data.append('chunk', chunk)
    const promise = new Promise<void>((resolve, reject) => {
      xhr.open('POST', apiUrl(`/upload/${taskId}/chunk`))
      xhr.withCredentials = true
      xhr.timeout = 10 * 60 * 1000
      xhr.upload.onprogress = (event) => onProgress(event.loaded)
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) resolve()
        else {
          let message = `分片上传失败 (${xhr.status})`
          try { message = JSON.parse(xhr.responseText).error || message } catch { /* noop */ }
          reject(new Error(message))
        }
      }
      xhr.onerror = () => reject(new Error('网络连接中断'))
      xhr.ontimeout = () => reject(new Error('分片上传超时'))
      xhr.onabort = () => reject(new DOMException('Upload paused', 'AbortError'))
      xhr.send(data)
    })
    return { promise, abort: () => xhr.abort() }
  },
}
