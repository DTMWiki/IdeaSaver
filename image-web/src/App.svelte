<script lang="ts">
  import { onMount } from 'svelte'
  import {
    ArchiveRestore, Check, ChevronRight, CircleAlert, Copy, Download, File as FileIcon, FileImage,
    Folder, FolderInput, FolderPlus, Grid2X2, HardDrive, Image, Link, List, LoaderCircle,
    LogIn, LogOut, Menu, MoreHorizontal, Move, Palette, PanelLeftClose, RefreshCw, Search, Share2,
    Trash2, Upload, X,
  } from 'lucide-svelte'
  import PublicShare from './components/PublicShare.svelte'
  import ImageThumbnail from './components/ImageThumbnail.svelte'
  import UploadPanel from './components/UploadPanel.svelte'
  import { authApi, filesApi, sharesApi } from './lib/api'
  import { uploadQueue } from './lib/upload-queue.svelte'
  import type { FileItem, Share, User, ViewMode, ViewName } from './lib/types'
  import { copyText, errorMessage, formatBytes, formatDate, isImage } from './lib/utils'

  type ModalName = 'mkdir' | 'rename' | 'move' | 'copy' | 'share' | 'delete' | 'preview' | null
  type Toast = { id: number; kind: 'ok' | 'error'; message: string }
  type Destination = { id: string; path: string }
  const themes = [
    { key: 'xuan', label: '宣纸' },
    { key: 'xuanye', label: '玄夜' },
    { key: 'tianshui', label: '天水' },
    { key: 'cangming', label: '沧溟' },
    { key: 'qiushan', label: '秋山' },
  ] as const
  type ThemeKey = typeof themes[number]['key']

  const shareCode = window.location.pathname.match(/^\/share\/([^/]+)/)?.[1]
  const isLoginCallback = /^\/login\/callback\/?$/.test(window.location.pathname)
  let booting = $state(true)
  let user = $state<User | null>(null)
  let authError = $state('')
  let view = $state<ViewName>('files')
  let viewMode = $state<ViewMode>('grid')
  let files = $state<FileItem[]>([])
  let trash = $state<FileItem[]>([])
  let shares = $state<Share[]>([])
  let directories = $state<Destination[]>([])
  let loading = $state(false)
  let sidebarOpen = $state(false)
  let currentId = $state<string | null>(null)
  let breadcrumbs = $state<{ id: string | null; name: string }[]>([{ id: null, name: '全部文件' }])
  let selected = $state<string[]>([])
  let query = $state('')
  let modal = $state<ModalName>(null)
  let target = $state<FileItem | null>(null)
  let formName = $state('')
  let destinationId = $state<string>('')
  let destinationsLoading = $state(false)
  let sharePassword = $state('')
  let shareExpiry = $state('604800')
  let shareResult = $state('')
  let previewObjectUrl = $state('')
  let theme = $state<ThemeKey>('xuan')
  let themePickerOpen = $state(false)
  let fileDragActive = $state(false)
  let submitting = $state(false)
  let toasts = $state<Toast[]>([])
  let fileInput = $state<HTMLInputElement>()
  let destinationRequest = 0

  let shownFiles = $derived(
    files
      .filter((file) => file.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()))
      .sort((a, b) => Number(b.is_directory) - Number(a.is_directory) || a.name.localeCompare(b.name, 'zh-CN')),
  )
  let allSelected = $derived(shownFiles.length > 0 && shownFiles.every((file) => selected.includes(file.id)))
  let quotaPercent = $derived(user?.storage_quota ? Math.min(100, user.storage_used / user.storage_quota * 100) : 0)

  function toast(message: string, kind: Toast['kind'] = 'ok'): void {
    const item = { id: Date.now() + Math.random(), kind, message }
    toasts.push(item)
    setTimeout(() => { toasts = toasts.filter((current) => current.id !== item.id) }, 3200)
  }

  async function loadAuth(): Promise<void> {
    booting = true
    try {
      user = await authApi.me()
      await refreshFiles()
      await uploadQueue.restore()
      uploadQueue.onComplete = () => { void refreshFiles() }
    } catch (cause) {
      authError = errorMessage(cause)
    } finally {
      booting = false
    }
  }

  async function completeLogin(): Promise<void> {
    booting = true
    authError = ''
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    const state = params.get('state')
    const providerError = params.get('error_description') || params.get('error')
    try {
      if (providerError) throw new Error(`登录授权失败：${providerError}`)
      if (!code) throw new Error('登录回调缺少授权码，请重新登录')
      user = await authApi.callback(code, state)
      window.history.replaceState({}, '', '/')
      await refreshFiles()
      await uploadQueue.restore()
      uploadQueue.onComplete = () => { void refreshFiles() }
    } catch (cause) {
      user = null
      authError = errorMessage(cause)
    } finally {
      booting = false
    }
  }

  async function refreshFiles(): Promise<void> {
    loading = true
    try {
      files = await filesApi.list(currentId)
      selected = []
    } catch (cause) { toast(errorMessage(cause), 'error') }
    finally { loading = false }
  }

  async function selectView(next: ViewName): Promise<void> {
    view = next
    sidebarOpen = false
    selected = []
    query = ''
    loading = true
    try {
      if (next === 'files') await refreshFiles()
      if (next === 'trash') trash = await filesApi.trash()
      if (next === 'shares') shares = await sharesApi.list()
    } catch (cause) { toast(errorMessage(cause), 'error') }
    finally { loading = false }
  }

  async function openDirectory(file: FileItem): Promise<void> {
    if (!file.is_directory) { await openPreview(file); return }
    currentId = file.id
    breadcrumbs.push({ id: file.id, name: file.name })
    await refreshFiles()
  }

  async function goBreadcrumb(index: number): Promise<void> {
    breadcrumbs = breadcrumbs.slice(0, index + 1)
    currentId = breadcrumbs[index].id
    await refreshFiles()
  }

  function toggleSelected(id: string): void {
    selected = selected.includes(id) ? selected.filter((item) => item !== id) : [...selected, id]
  }

  function toggleAll(): void {
    selected = allSelected ? [] : shownFiles.map((file) => file.id)
  }

  async function loadDestinations(): Promise<void> {
    const requestId = ++destinationRequest
    const blocked = new Set(target ? [target.id] : selected)
    const visited = new Set<string>()
    const collected: Destination[] = []
    destinationsLoading = true
    directories = []

    async function visit(parentId: string | null, ancestors: string[]): Promise<void> {
      const children = await filesApi.list(parentId)
      const childDirectories = children
        .filter((file) => file.is_directory)
        .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
      for (const directory of childDirectories) {
        if (visited.has(directory.id) || blocked.has(directory.id)) continue
        visited.add(directory.id)
        const names = [...ancestors, directory.name]
        collected.push({ id: directory.id, path: names.join(' / ') })
        await visit(directory.id, names)
      }
    }

    try {
      await visit(null, [])
      if (requestId === destinationRequest) directories = collected
    } catch (cause) {
      if (requestId === destinationRequest) toast(`读取目标目录失败：${errorMessage(cause)}`, 'error')
    } finally {
      if (requestId === destinationRequest) destinationsLoading = false
    }
  }

  function openModal(name: ModalName, file: FileItem | null = null): void {
    target = file
    modal = name
    formName = name === 'rename' && file ? file.name : ''
    destinationId = ''
    directories = []
    destinationsLoading = false
    sharePassword = ''
    shareExpiry = '604800'
    shareResult = ''
    if (name === 'move' || name === 'copy') void loadDestinations()
  }

  async function openPreview(file: FileItem): Promise<void> {
    URL.revokeObjectURL(previewObjectUrl)
    previewObjectUrl = ''
    openModal('preview', file)
    try {
      previewObjectUrl = URL.createObjectURL(await filesApi.preview(file.id))
    } catch (cause) {
      toast(errorMessage(cause), 'error')
    }
  }

  function closeModal(): void {
    if (submitting) return
    destinationRequest += 1
    destinationsLoading = false
    modal = null
    target = null
  }

  async function submitModal(): Promise<void> {
    submitting = true
    try {
      if (modal === 'mkdir') await filesApi.mkdir(formName.trim(), currentId)
      if (modal === 'rename' && target) await filesApi.rename(target.id, formName.trim())
      if (modal === 'delete') {
        const ids = target ? [target.id] : selected
        if (ids.length === 1) await filesApi.softDelete(ids[0])
        else {
          const result = await filesApi.batchDelete(ids)
          if (!result.ok) {
            const failedCount = result.failed?.length || Math.max(0, ids.length - result.deleted)
            toast(`已删除 ${result.deleted} 项，${failedCount} 项失败`, 'error')
            modal = null
            target = null
            await refreshFiles()
            return
          }
        }
      }
      if ((modal === 'move' || modal === 'copy')) {
        const ids = target ? [target.id] : selected
        if (modal === 'move') await Promise.all(ids.map((id) => filesApi.move(id, destinationId || null)))
        else await Promise.all(ids.map((id) => filesApi.copy(id, destinationId || null)))
      }
      if (modal === 'share' && target) {
        const result = await filesApi.createShare(target.id, sharePassword || undefined, Number(shareExpiry) || undefined)
        shareResult = `${window.location.origin}/share/${result.code}`
        await copyText(shareResult).catch(() => undefined)
        toast('分享已创建，链接已复制')
        submitting = false
        return
      }
      toast(modal === 'delete' ? '已移入回收站' : '操作已完成')
      modal = null
      target = null
      await refreshFiles()
    } catch (cause) { toast(errorMessage(cause), 'error') }
    finally { submitting = false }
  }

  async function copyLink(file: FileItem): Promise<void> {
    try {
      const result = await filesApi.link(file.id)
      await copyText(result.url)
      toast('文件链接已复制')
    } catch (cause) { toast(errorMessage(cause), 'error') }
  }

  async function handleFiles(fileList: FileList | File[]): Promise<void> {
    const list = Array.from(fileList)
    if (list.length) await uploadQueue.add(list, currentId)
    if (fileInput) fileInput.value = ''
  }

  function acceptsFileDrop(event: DragEvent): boolean {
    return view === 'files' && Boolean(event.dataTransfer?.types.includes('Files'))
  }

  function handleWindowDragOver(event: DragEvent): void {
    if (!acceptsFileDrop(event)) return
    event.preventDefault()
    event.dataTransfer!.dropEffect = 'copy'
    fileDragActive = true
  }

  function handleWindowDragLeave(event: DragEvent): void {
    if (event.relatedTarget === null) fileDragActive = false
  }

  function handleWindowDrop(event: DragEvent): void {
    if (!acceptsFileDrop(event)) return
    event.preventDefault()
    fileDragActive = false
    if (event.dataTransfer?.files.length) void handleFiles(event.dataTransfer.files)
  }

  async function restore(file: FileItem): Promise<void> {
    try { await filesApi.restore(file.id); trash = await filesApi.trash(); toast('文件已恢复') }
    catch (cause) { toast(errorMessage(cause), 'error') }
  }

  async function destroy(file: FileItem): Promise<void> {
    if (!confirm(`永久删除「${file.name}」？此操作无法撤销。`)) return
    try { await filesApi.permanentDelete(file.id); trash = await filesApi.trash(); toast('已永久删除') }
    catch (cause) { toast(errorMessage(cause), 'error') }
  }

  async function deleteShare(share: Share): Promise<void> {
    if (!confirm('删除后原分享链接会立即失效，继续吗？')) return
    try { await sharesApi.remove(share.id); shares = await sharesApi.list(); toast('分享已删除') }
    catch (cause) { toast(errorMessage(cause), 'error') }
  }

  async function login(): Promise<void> {
    try { window.location.href = await authApi.login() }
    catch (cause) { toast(errorMessage(cause), 'error') }
  }

  async function logout(): Promise<void> {
    await authApi.logout().catch(() => undefined)
    user = null
  }

  function setTheme(next: ThemeKey): void {
    theme = next
    document.documentElement.dataset.theme = next
    localStorage.setItem('ideasaver-theme', next)
    themePickerOpen = false
  }

  onMount(() => {
    const saved = localStorage.getItem('ideasaver-theme')
    theme = themes.some((candidate) => candidate.key === saved) ? saved as ThemeKey : 'xuan'
    document.documentElement.dataset.theme = theme
    if (!shareCode) {
      if (isLoginCallback) void completeLogin()
      else void loadAuth()
    }
    window.addEventListener('dragover', handleWindowDragOver)
    window.addEventListener('dragleave', handleWindowDragLeave)
    window.addEventListener('drop', handleWindowDrop)
    return () => {
      window.removeEventListener('dragover', handleWindowDragOver)
      window.removeEventListener('dragleave', handleWindowDragLeave)
      window.removeEventListener('drop', handleWindowDrop)
    }
  })
</script>

{#if shareCode}
  <PublicShare code={decodeURIComponent(shareCode)} />
{:else if booting}
  <main class="center-state"><LoaderCircle class="spin" size={28} /><p>正在连接 DTMwiki…</p></main>
{:else if !user}
  <main class="login-page">
    <section class="login-sheet">
      <div class="brand-mark"><Image size={24} strokeWidth={2} /></div>
      <p class="eyebrow">DTMwiki · 音乐制作图片托管</p>
      <h1>整理、分享与传递文件</h1>
      <p>使用 DTMwiki 账号登录。认证由现有服务处理，新前端不保存访问令牌。</p>
      {#if authError}<div class="notice error"><CircleAlert size={17} /> {authError}</div>{/if}
      <button class="primary-button" onclick={login}><LogIn size={17} /> 登录 DTMwiki</button>
    </section>
  </main>
{:else}
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <button class="icon-button mobile-only" title="打开导航" onclick={() => sidebarOpen = !sidebarOpen}><Menu size={20} /></button>
        <span class="brand-mark small"><Image size={16} strokeWidth={2} /></span>
        <div><strong>DTMwiki</strong><span>图床</span></div>
      </div>
      <div class="topbar-actions">
        <details class="theme-picker" bind:open={themePickerOpen}>
          <summary class="icon-button" title="选择主题" aria-label="选择主题"><Palette size={18} /></summary>
          <div class="theme-menu" role="menu" aria-label="主题选择">
            {#each themes as option}
              <button class:active={theme === option.key} data-theme-choice={option.key} role="menuitemradio" aria-checked={theme === option.key} onclick={() => setTheme(option.key)}>
                <span class="theme-swatch"></span>{option.label}
              </button>
            {/each}
          </div>
        </details>
        <span class="user-identity">{user.display_name || user.username}</span>
        <button class="icon-button" title="退出登录" aria-label="退出登录" onclick={logout}><LogOut size={16} /></button>
      </div>
    </header>

    <aside class:open={sidebarOpen} class="sidebar">
      <button class:active={view === 'files'} onclick={() => selectView('files')}><HardDrive size={18} />全部文件</button>
      <button class:active={view === 'shares'} onclick={() => selectView('shares')}><Share2 size={18} />我的分享</button>
      <button class:active={view === 'trash'} onclick={() => selectView('trash')}><Trash2 size={18} />回收站</button>
      <div class="sidebar-spacer"></div>
      <div class="quota">
        <div><span>存储空间</span><span>{Math.round(quotaPercent)}%</span></div>
        <progress max="100" value={quotaPercent}></progress>
        <small>{formatBytes(user.storage_used)} / {formatBytes(user.storage_quota)}</small>
      </div>
    </aside>
    {#if sidebarOpen}<button class="sidebar-scrim" aria-label="关闭导航" onclick={() => sidebarOpen = false}></button>{/if}

    <main class="workspace">
      {#if view === 'files'}
        <div class="page-heading">
          <div>
            <p class="eyebrow">个人空间</p>
            <h1>{breadcrumbs[breadcrumbs.length - 1]?.name}</h1>
          </div>
          <div class="heading-actions">
            <button class="secondary-button" onclick={() => openModal('mkdir')}><FolderPlus size={17} />新建文件夹</button>
            <button class="primary-button" onclick={() => fileInput?.click()}><Upload size={17} />上传文件</button>
          </div>
        </div>

        <div class="toolbar">
          <nav class="breadcrumbs" aria-label="当前位置">
            {#each breadcrumbs as crumb, index}
              {#if index > 0}<ChevronRight size={14} />{/if}
              <button onclick={() => goBreadcrumb(index)}>{crumb.name}</button>
            {/each}
          </nav>
          <div class="toolbar-tools">
            <label class="search"><Search size={16} /><input bind:value={query} placeholder="搜索当前目录" /></label>
            <button class="icon-button" title="刷新" onclick={refreshFiles}><RefreshCw size={17} /></button>
            <div class="segmented" aria-label="视图方式">
              <button class:active={viewMode === 'grid'} title="网格视图" onclick={() => viewMode = 'grid'}><Grid2X2 size={16} /></button>
              <button class:active={viewMode === 'table'} title="列表视图" onclick={() => viewMode = 'table'}><List size={17} /></button>
            </div>
          </div>
        </div>

        {#if selected.length > 0}
          <div class="selection-bar">
            <strong>已选择 {selected.length} 项</strong>
            <button onclick={() => openModal('move')}><Move size={16} />移动</button>
            <button onclick={() => openModal('copy')}><Copy size={16} />复制</button>
            <button class="danger-text" onclick={() => openModal('delete')}><Trash2 size={16} />删除</button>
            <button class="icon-button" title="清除选择" onclick={() => selected = []}><X size={16} /></button>
          </div>
        {/if}

        <input class="visually-hidden" bind:this={fileInput} type="file" multiple onchange={(event) => handleFiles(event.currentTarget.files || [])} />

        {#if loading}
          <div class="center-state compact"><LoaderCircle class="spin" size={24} /><p>正在读取文件…</p></div>
        {:else if shownFiles.length === 0}
          <button class="empty-drop" onclick={() => fileInput?.click()}>
            <Upload size={28} /><strong>{query ? '没有匹配的文件' : '拖入文件，开始上传'}</strong>
            <span>{query ? '尝试更换搜索内容' : '也可以点击这里选择多个文件'}</span>
          </button>
        {:else if viewMode === 'grid'}
          <div class="file-grid">
            {#each shownFiles as file (file.id)}
              <article class:selected={selected.includes(file.id)} class="file-card">
                <label class="file-check" title="选择">
                  <input type="checkbox" checked={selected.includes(file.id)} onchange={() => toggleSelected(file.id)} />
                  <span><Check size={13} /></span>
                </label>
                <button class="thumbnail" type="button" aria-label={`打开 ${file.name}`} ondblclick={() => void openDirectory(file)}>
                    {#if file.is_directory}<Folder size={50} strokeWidth={1.35} />
                    {:else if isImage(file)}<ImageThumbnail id={file.id} name={file.name} />
                    {:else}<FileIcon size={44} strokeWidth={1.35} />{/if}
                </button>
                <div class="card-main">
                  <div class="file-meta"><strong title={file.name}>{file.name}</strong><span>{file.is_directory ? '文件夹' : formatBytes(file.size)} · {formatDate(file.updated_at)}</span></div>
                </div>
                <div class="card-actions">
                  {#if !file.is_directory}<button title="复制链接" onclick={() => copyLink(file)}><Link size={15} /></button>{/if}
                  <button title="更多操作" onclick={(event) => { const details = event.currentTarget.nextElementSibling as HTMLDetailsElement; details.open = !details.open }}><MoreHorizontal size={16} /></button>
                  <details class="action-menu"><summary>操作</summary>
                    <button onclick={() => void openDirectory(file)}>{file.is_directory ? '打开' : '预览'}</button>
                    {#if !file.is_directory}<button onclick={() => openModal('share', file)}>创建分享</button>{/if}
                    <button onclick={() => openModal('rename', file)}>重命名</button>
                    <button onclick={() => openModal('move', file)}>移动</button>
                    <button onclick={() => openModal('copy', file)}>复制</button>
                    <button class="danger-text" onclick={() => openModal('delete', file)}>移入回收站</button>
                  </details>
                </div>
              </article>
            {/each}
          </div>
        {:else}
          <div class="file-table-wrap">
            <table class="file-table">
              <thead><tr><th class="check-cell"><input type="checkbox" checked={allSelected} onchange={toggleAll} /></th><th>名称</th><th>大小</th><th>修改时间</th><th></th></tr></thead>
              <tbody>{#each shownFiles as file (file.id)}<tr class:selected={selected.includes(file.id)}>
                <td class="check-cell"><input type="checkbox" checked={selected.includes(file.id)} onchange={() => toggleSelected(file.id)} /></td>
                <td><button class="file-name" onclick={() => openDirectory(file)}>{#if file.is_directory}<Folder size={20} />{:else if isImage(file)}<FileImage size={20} />{:else}<FileIcon size={20} />{/if}<span>{file.name}</span></button></td>
                <td>{file.is_directory ? '-' : formatBytes(file.size)}</td><td>{formatDate(file.updated_at)}</td>
                <td class="row-actions">{#if !file.is_directory}<button title="复制链接" onclick={() => copyLink(file)}><Link size={16} /></button>{/if}<button title="删除" onclick={() => openModal('delete', file)}><Trash2 size={16} /></button></td>
              </tr>{/each}</tbody>
            </table>
          </div>
        {/if}
      {:else if view === 'trash'}
        <div class="page-heading"><div><p class="eyebrow">30 天内可恢复</p><h1>回收站</h1></div></div>
        <div class="list-panel">
          {#if loading}<div class="center-state compact"><LoaderCircle class="spin" size={24} /></div>
          {:else if trash.length === 0}<div class="empty-inline"><Trash2 size={30} /><strong>回收站为空</strong></div>
          {:else}{#each trash as file (file.id)}<article class="list-row">
            <div class="list-icon">{#if file.is_directory}<Folder size={22} />{:else}<FileIcon size={22} />{/if}</div>
            <div class="list-copy"><strong>{file.name}</strong><span>{file.is_directory ? '文件夹' : formatBytes(file.size)} · 删除于 {formatDate(file.deleted_at)}</span></div>
            <div class="row-actions"><button title="恢复" onclick={() => restore(file)}><ArchiveRestore size={17} /></button><button class="danger-text" title="永久删除" onclick={() => destroy(file)}><Trash2 size={17} /></button></div>
          </article>{/each}{/if}
        </div>
      {:else}
        <div class="page-heading"><div><p class="eyebrow">外部访问</p><h1>我的分享</h1></div></div>
        <div class="list-panel">
          {#if loading}<div class="center-state compact"><LoaderCircle class="spin" size={24} /></div>
          {:else if shares.length === 0}<div class="empty-inline"><Share2 size={30} /><strong>还没有分享链接</strong><button class="text-button" onclick={() => selectView('files')}>去文件页创建</button></div>
          {:else}{#each shares as share (share.id)}<article class="list-row">
            <div class="list-icon"><Share2 size={21} /></div>
            <div class="list-copy"><strong>{share.file_name || '文件已删除'}</strong><span>{share.has_password ? '密码保护 · ' : ''}{share.expires_at ? `有效至 ${formatDate(share.expires_at)}` : '永久有效'} · {share.view_count} 次访问</span></div>
            <div class="row-actions"><button title="复制分享链接" onclick={() => { void copyText(`${window.location.origin}/share/${share.code}`); toast('分享链接已复制') }}><Copy size={17} /></button><a title="打开分享" href={`/share/${share.code}`} target="_blank"><Link size={17} /></a><button class="danger-text" title="删除分享" onclick={() => deleteShare(share)}><Trash2 size={17} /></button></div>
          </article>{/each}{/if}
        </div>
      {/if}
    </main>
    {#if fileDragActive && view === 'files'}
      <div class="file-drop-overlay" aria-hidden="true"><div><Upload size={30} /><strong>释放以上传文件</strong><span>文件将上传至当前目录</span></div></div>
    {/if}
  </div>

  <UploadPanel />

  {#if modal}
    <div class="modal-backdrop" role="presentation" onclick={(event) => { if (event.target === event.currentTarget) closeModal() }}>
      <section class:preview-modal={modal === 'preview'} class="modal" role="dialog" aria-modal="true">
        <header><div><p class="eyebrow">{modal === 'preview' ? '文件预览' : '文件操作'}</p><h2>
          {modal === 'mkdir' ? '新建文件夹' : modal === 'rename' ? '重命名' : modal === 'move' ? '移动到' : modal === 'copy' ? '复制到' : modal === 'share' ? '创建分享' : modal === 'delete' ? '移入回收站' : target?.name}
        </h2></div><button class="icon-button" title="关闭" onclick={closeModal}><X size={18} /></button></header>
        {#if modal === 'preview' && target}
          <div class="preview-area">
            {#if previewObjectUrl}
              {#if isImage(target)}<img src={previewObjectUrl} alt={target.name} />{:else}<iframe src={previewObjectUrl} title={target.name}></iframe>{/if}
            {:else}<LoaderCircle class="spin" size={28} />{/if}
          </div>
          <footer><button class="secondary-button" onclick={() => copyLink(target!)}><Link size={16} />复制链接</button>{#if previewObjectUrl}<a class="primary-button" href={previewObjectUrl} download={target.name}><Download size={16} />下载</a>{/if}</footer>
        {:else}
          <div class="modal-body">
            {#if modal === 'mkdir' || modal === 'rename'}
              <label>{modal === 'mkdir' ? '文件夹名称' : '新名称'}<input bind:value={formName} onkeydown={(event) => { if (event.key === 'Enter' && formName.trim()) void submitModal() }} /></label>
            {:else if modal === 'move' || modal === 'copy'}
              <p class="muted">{target ? `“${target.name}”` : `${selected.length} 个所选项目`}将被{modal === 'move' ? '移动' : '复制'}到：</p>
              <label>目标目录<select bind:value={destinationId} disabled={destinationsLoading}><option value="">{destinationsLoading ? '正在读取目录…' : '全部文件（根目录）'}</option>{#each directories as directory}<option value={directory.id}>{directory.path}</option>{/each}</select></label>
            {:else if modal === 'share'}
              {#if shareResult}
                <div class="notice success"><Check size={17} /><span><strong>分享已创建</strong><small>{shareResult}</small></span></div>
              {:else}
                <label>访问密码（可选）<input type="password" bind:value={sharePassword} placeholder="留空则无需密码" /></label>
                <label>有效期<select bind:value={shareExpiry}><option value="3600">1 小时</option><option value="86400">1 天</option><option value="604800">7 天</option><option value="2592000">30 天</option><option value="0">永久</option></select></label>
              {/if}
            {:else if modal === 'delete'}
              <div class="notice warning"><CircleAlert size={19} /><span><strong>确认移入回收站？</strong><small>{target ? target.name : `${selected.length} 个所选项目`}可在 30 天内恢复。</small></span></div>
            {/if}
          </div>
          <footer><button class="secondary-button" onclick={closeModal}>{shareResult ? '完成' : '取消'}</button>{#if !shareResult}<button class:danger-button={modal === 'delete'} class="primary-button" disabled={submitting || destinationsLoading || ((modal === 'mkdir' || modal === 'rename') && !formName.trim())} onclick={submitModal}>{#if submitting}<LoaderCircle class="spin" size={16} />{/if}{modal === 'share' ? '创建并复制链接' : modal === 'delete' ? '确认删除' : '确认'}</button>{/if}</footer>
        {/if}
      </section>
    </div>
  {/if}

  <div class="toast-stack" aria-live="polite">{#each toasts as item (item.id)}<div class:error={item.kind === 'error'} class="toast">{#if item.kind === 'ok'}<Check size={16} />{:else}<CircleAlert size={16} />{/if}{item.message}</div>{/each}</div>
{/if}
