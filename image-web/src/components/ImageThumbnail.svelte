<script module lang="ts">
  import { filesApi } from '../lib/api'

  const cache = new Map<string, string>()
  const pending = new Map<string, Promise<string>>()
  const cacheLimit = 48

  async function getThumbnailUrl(id: string): Promise<string> {
    const cached = cache.get(id)
    if (cached) {
      cache.delete(id)
      cache.set(id, cached)
      return cached
    }

    const active = pending.get(id)
    if (active) return active

    const request = filesApi.thumbnail(id).catch(() => filesApi.preview(id)).then((blob) => {
      const url = URL.createObjectURL(blob)
      cache.set(id, url)
      if (cache.size > cacheLimit) {
        const oldest = cache.entries().next().value as [string, string]
        cache.delete(oldest[0])
        URL.revokeObjectURL(oldest[1])
      }
      return url
    }).finally(() => pending.delete(id))

    pending.set(id, request)
    return request
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte'
  import { FileImage, LoaderCircle } from 'lucide-svelte'

  interface Props {
    id: string
    name: string
  }

  let { id, name }: Props = $props()
  let host = $state<HTMLSpanElement>()
  let url = $state('')
  let failed = $state(false)

  async function load(): Promise<void> {
    try {
      url = await getThumbnailUrl(id)
    } catch {
      failed = true
    }
  }

  onMount(() => {
    const observer = new IntersectionObserver((entries) => {
      if (!entries[0]?.isIntersecting) return
      observer.disconnect()
      void load()
    }, { rootMargin: '240px 0px' })

    if (host) observer.observe(host)
    return () => observer.disconnect()
  })
</script>

<span class="image-thumbnail" bind:this={host}>
  {#if url}
    <img src={url} alt={name} loading="lazy" decoding="async" />
  {:else if failed}
    <FileImage size={44} strokeWidth={1.35} aria-label="图片缩略图加载失败" />
  {:else}
    <LoaderCircle class="spin" size={22} aria-label="正在加载缩略图" />
  {/if}
</span>

<style>
  .image-thumbnail { width: 100%; height: 100%; display: grid; place-items: center; }
  img { width: 100%; height: 100%; object-fit: cover; }
</style>
