<script lang="ts">
  import { CircleAlert, Download, File as FileIcon, Image, KeyRound, LoaderCircle, RefreshCw } from 'lucide-svelte'
  import { onMount } from 'svelte'
  import { ApiError, sharesApi } from '../lib/api'
  import { errorMessage, formatBytes } from '../lib/utils'

  interface Props { code: string }
  type PageState = 'loading' | 'password' | 'ready' | 'error'
  const terminalCodes = new Set(['password_required', 'password_invalid', 'share_expired', 'share_not_found', 'share_banned', 'share_token_invalid'])

  let { code }: Props = $props()
  let password = $state('')
  let loading = $state(false)
  let error = $state('')
  let errorTitle = $state('无法打开分享')
  let pageState = $state<PageState>('loading')
  let result = $state<Awaited<ReturnType<typeof sharesApi.access>> | null>(null)
  let unlockedPassword: string | undefined
  let refreshTimer: number | undefined
  let active = false

  function clearRefreshTimer(): void {
    if (refreshTimer !== undefined) window.clearTimeout(refreshTimer)
    refreshTimer = undefined
  }

  function applyAccessResult(
    next: Awaited<ReturnType<typeof sharesApi.access>>,
    usedPassword?: string,
  ): void {
    result = next
    unlockedPassword = usedPassword
    error = ''
    pageState = 'ready'
    clearRefreshTimer()
    if (!active) return
    const ttl = Math.max(30, Number(next.token_expires_in) || 900)
    const refreshLead = Math.min(60, Math.max(5, Math.floor(ttl / 5)))
    const refreshAfter = (ttl - refreshLead) * 1000
    refreshTimer = window.setTimeout(() => { void refreshAccess() }, refreshAfter)
  }

  async function refreshAccess(): Promise<void> {
    try {
      applyAccessResult(await sharesApi.access(code, unlockedPassword), unlockedPassword)
    } catch (cause) {
      clearRefreshTimer()
      if (cause instanceof ApiError && cause.code && terminalCodes.has(cause.code)) {
        showAccessError(cause)
        return
      }
      if (active) refreshTimer = window.setTimeout(() => { void refreshAccess() }, 15_000)
    }
  }

  function showAccessError(cause: unknown): void {
    clearRefreshTimer()
    result = null
    const code = cause instanceof ApiError ? cause.code : undefined
    if (code === 'password_required' || code === 'password_invalid') {
      pageState = 'password'
      unlockedPassword = undefined
      error = code === 'password_invalid' ? '密码错误，请重试' : ''
      return
    }
    pageState = 'error'
    unlockedPassword = undefined
    if (code === 'share_expired') errorTitle = '分享已过期'
    else if (code === 'share_not_found') errorTitle = '分享不存在'
    else if (code === 'share_banned') errorTitle = '文件不可访问'
    else if (code === 'share_token_invalid') errorTitle = '下载凭证已失效'
    else errorTitle = '无法打开分享'
    error = errorMessage(cause)
  }

  async function unlock(): Promise<void> {
    loading = true
    error = ''
    if (pageState !== 'password') pageState = 'loading'
    const providedPassword = password.trim() || undefined
    try { applyAccessResult(await sharesApi.access(code, providedPassword), providedPassword) }
    catch (cause) { showAccessError(cause) }
    finally { loading = false }
  }

  onMount(() => {
    active = true
    void unlock()
    return () => {
      active = false
      clearRefreshTimer()
    }
  })
</script>

<main class="share-page">
  <section class="share-sheet">
    <div class="brand-mark"><Image size={24} strokeWidth={2} /></div>
    <p class="eyebrow">DTMwiki · 音乐制作社区文件分享</p>
    {#if pageState === 'ready' && result}
      <div class="share-file-icon">{#if result.file.is_image}<Image size={34} />{:else}<FileIcon size={34} />{/if}</div>
      <h1>{result.file.name}</h1>
      <p class="muted">{formatBytes(result.file.size)} · 此下载链接将在短时间后失效</p>
      {#if result.file.is_image}
        <img class="share-preview" src={sharesApi.downloadUrl(code, result.download_token, true)} alt={result.file.name} />
      {/if}
      <a class="primary-button share-download" href={sharesApi.downloadUrl(code, result.download_token)}>
        <Download size={17} /> 下载文件
      </a>
    {:else if pageState === 'password'}
      <KeyRound size={36} class="share-key" />
      <h1>需要访问密码</h1>
      <p class="muted">输入分享密码后即可查看和下载文件。</p>
      <form onsubmit={(event) => { event.preventDefault(); void unlock() }}>
        <label for="share-password">访问密码</label>
        <input id="share-password" type="password" bind:value={password} placeholder="请输入访问密码" />
        {#if error}<p class="form-error">{error}</p>{/if}
        <button class="primary-button" disabled={loading}>
          {#if loading}<LoaderCircle class="spin" size={17} />{/if} 打开分享
        </button>
      </form>
    {:else if pageState === 'error'}
      <CircleAlert size={36} class="share-error" />
      <h1>{errorTitle}</h1>
      <p class="muted">{error}</p>
      <button class="secondary-button share-retry" onclick={() => { void unlock() }}><RefreshCw size={17} />重新检查</button>
    {:else}
      <LoaderCircle size={32} class="share-key spin" />
      <h1>正在打开分享</h1>
      <p class="muted">正在检查链接和访问权限…</p>
    {/if}
  </section>
</main>
