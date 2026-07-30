import { uploadApi } from './api'
import type { ServerUploadTask, UploadTask } from './types'

const STORAGE_KEY = 'ideasaver-image-upload-tasks-v1'
const DB_NAME = 'ideasaver-image-uploads'
const DB_STORE = 'files'
const concurrency = Math.max(1, Math.min(6, Number(import.meta.env.VITE_UPLOAD_CONCURRENCY) || 3))

type TaskSnapshot = Omit<UploadTask, 'file'>

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1)
    request.onupgradeneeded = () => request.result.createObjectStore(DB_STORE)
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

async function storeFile(id: string, file: File): Promise<void> {
  const db = await openDb()
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(DB_STORE, 'readwrite')
    tx.objectStore(DB_STORE).put(file, id)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
  db.close()
}

async function readFile(id: string): Promise<File | undefined> {
  const db = await openDb()
  const file = await new Promise<File | undefined>((resolve, reject) => {
    const request = db.transaction(DB_STORE).objectStore(DB_STORE).get(id)
    request.onsuccess = () => resolve(request.result as File | undefined)
    request.onerror = () => reject(request.error)
  })
  db.close()
  return file
}

async function deleteFile(id: string): Promise<void> {
  const db = await openDb()
  const tx = db.transaction(DB_STORE, 'readwrite')
  tx.objectStore(DB_STORE).delete(id)
  db.close()
}

function serialize(task: UploadTask): TaskSnapshot {
  const { file: _file, ...snapshot } = task
  return snapshot
}

class UploadQueue {
  tasks = $state<UploadTask[]>([])
  visible = $state(false)
  private active = 0
  private requests = new Map<string, () => void>()
  onComplete: (() => void) | undefined

  get pendingCount(): number {
    return this.tasks.filter((task) => ['pending', 'uploading', 'paused'].includes(task.status)).length
  }

  async restore(): Promise<void> {
    let snapshots: TaskSnapshot[] = []
    try { snapshots = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]') } catch { /* noop */ }
    const serverTasks = await uploadApi.tasks().catch(() => [] as ServerUploadTask[])
    const restored: UploadTask[] = []

    for (const snapshot of snapshots) {
      if (snapshot.status === 'completed') continue
      const file = await readFile(snapshot.localId).catch(() => undefined)
      if (!file) continue
      const server = snapshot.serverId ? serverTasks.find((task) => task.id === snapshot.serverId) : undefined
      const uploadedChunks = server?.uploaded_chunks ?? snapshot.uploadedChunks
      const chunkSize = server?.chunk_size ?? snapshot.chunkSize
      const uploadedSize = Math.min(file.size, server?.uploaded_size ?? uploadedChunks * chunkSize)
      restored.push({
        ...snapshot,
        file,
        status: snapshot.status === 'failed' ? 'failed' : 'paused',
        uploadedChunks,
        uploadedSize,
        chunkSize,
        totalChunks: server?.total_chunks ?? snapshot.totalChunks,
        progress: file.size ? Math.round(uploadedSize / file.size * 100) : 0,
        speed: 0,
        error: snapshot.status === 'failed' ? snapshot.error : '上传已中断，可继续上传',
      })
    }
    this.tasks = restored
    this.visible = restored.length > 0
    this.persist()
  }

  async add(files: File[], parentId: string | null): Promise<void> {
    const created = files.map((file, index): UploadTask => ({
      localId: `upload-${Date.now()}-${index}-${crypto.randomUUID()}`,
      file,
      parentId,
      filename: file.name,
      totalSize: file.size,
      uploadedSize: 0,
      chunkSize: 0,
      totalChunks: 0,
      uploadedChunks: 0,
      status: 'pending',
      progress: 0,
      speed: 0,
    }))
    this.tasks.push(...created)
    this.visible = true
    await Promise.all(created.map((task) => storeFile(task.localId, task.file).catch(() => undefined)))
    this.persist()
    this.process()
  }

  async pause(localId: string): Promise<void> {
    const task = this.tasks.find((item) => item.localId === localId)
    if (!task || !['pending', 'uploading'].includes(task.status)) return
    task.status = 'paused'
    task.speed = 0
    task.error = undefined
    this.requests.get(localId)?.()
    if (task.serverId) await uploadApi.pause(task.serverId).catch(() => undefined)
    this.persist()
  }

  async resume(localId: string): Promise<void> {
    const task = this.tasks.find((item) => item.localId === localId)
    if (!task || !['paused', 'failed'].includes(task.status)) return
    if (task.serverId) await uploadApi.resume(task.serverId).catch(() => undefined)
    task.status = 'pending'
    task.error = undefined
    this.persist()
    this.process()
  }

  async remove(localId: string): Promise<void> {
    this.requests.get(localId)?.()
    this.tasks = this.tasks.filter((task) => task.localId !== localId)
    await deleteFile(localId).catch(() => undefined)
    this.persist()
  }

  async clearCompleted(): Promise<void> {
    const done = this.tasks.filter((task) => task.status === 'completed')
    this.tasks = this.tasks.filter((task) => task.status !== 'completed')
    await Promise.all(done.map((task) => deleteFile(task.localId).catch(() => undefined)))
    this.persist()
  }

  private persist(): void {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(this.tasks.map(serialize)))
  }

  private update(localId: string, values: Partial<UploadTask>): UploadTask | undefined {
    const task = this.tasks.find((item) => item.localId === localId)
    if (task) Object.assign(task, values)
    this.persist()
    return task
  }

  private process(): void {
    while (this.active < concurrency) {
      const task = this.tasks.find((item) => item.status === 'pending')
      if (!task) break
      this.active += 1
      task.status = 'uploading'
      void this.run(task.localId).finally(() => {
        this.active -= 1
        this.process()
      })
    }
  }

  private async run(localId: string): Promise<void> {
    let task = this.tasks.find((item) => item.localId === localId)
    if (!task) return
    try {
      if (!task.serverId) {
        const result = await uploadApi.init(task.file, task.parentId)
        task = this.update(localId, {
          serverId: result.task_id,
          chunkSize: result.chunk_size,
          totalChunks: result.total_chunks,
        })
      }
      if (!task?.serverId) throw new Error('上传任务初始化失败')
      const serverId = task.serverId

      let uploadedBytes = task.uploadedSize
      const startedAt = Date.now()
      for (let index = task.uploadedChunks; index < task.totalChunks; index += 1) {
        task = this.tasks.find((item) => item.localId === localId)
        if (!task || task.status !== 'uploading') return
        const start = index * task.chunkSize
        const end = Math.min(start + task.chunkSize, task.totalSize)
        const transfer = uploadApi.uploadChunk(serverId, index, task.file.slice(start, end), (loaded) => {
          const current = Math.min(uploadedBytes + loaded, task!.totalSize)
          const seconds = Math.max((Date.now() - startedAt) / 1000, 0.1)
          this.update(localId, {
            uploadedSize: current,
            progress: Math.round(current / task!.totalSize * 100),
            speed: current / seconds,
          })
        })
        this.requests.set(localId, transfer.abort)
        await transfer.promise
        this.requests.delete(localId)
        uploadedBytes = Math.min((index + 1) * task.chunkSize, task.totalSize)
        this.update(localId, {
          uploadedChunks: index + 1,
          uploadedSize: uploadedBytes,
          progress: Math.round(uploadedBytes / task.totalSize * 100),
        })
      }

      const result = await uploadApi.complete(serverId, task.parentId)
      this.update(localId, {
        status: 'completed', progress: 100, uploadedSize: task.totalSize, speed: 0,
        url: result.url, markdown: result.markdown,
      })
      this.onComplete?.()
    } catch (error) {
      this.requests.delete(localId)
      task = this.tasks.find((item) => item.localId === localId)
      if (!task || task.status === 'paused' || (error instanceof DOMException && error.name === 'AbortError')) return
      this.update(localId, {
        status: 'failed', speed: 0,
        error: error instanceof Error ? error.message : '上传失败',
      })
    }
  }
}

export const uploadQueue = new UploadQueue()
