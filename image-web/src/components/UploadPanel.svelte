<script lang="ts">
  import { CheckCircle2, ChevronDown, ChevronUp, Pause, Play, RotateCcw, UploadCloud, X } from 'lucide-svelte'
  import { uploadQueue } from '../lib/upload-queue.svelte'
  import { formatBytes } from '../lib/utils'

  let collapsed = $state(false)
</script>

{#if uploadQueue.visible && uploadQueue.tasks.length > 0}
  <aside class:collapsed class="upload-panel" aria-label="上传队列">
    <header>
      <div class="upload-title">
        <UploadCloud size={17} />
        <strong>上传队列</strong>
        <span>{uploadQueue.pendingCount ? `${uploadQueue.pendingCount} 项进行中` : '全部完成'}</span>
      </div>
      <div class="icon-row">
        {#if uploadQueue.tasks.some((task) => task.status === 'completed')}
          <button class="icon-button" title="清除已完成" onclick={() => uploadQueue.clearCompleted()}><CheckCircle2 size={17} /></button>
        {/if}
        <button class="icon-button" title={collapsed ? '展开队列' : '收起队列'} onclick={() => collapsed = !collapsed}>
          {#if collapsed}<ChevronUp size={17} />{:else}<ChevronDown size={17} />{/if}
        </button>
      </div>
    </header>

    {#if !collapsed}<div class="upload-list">
      {#each uploadQueue.tasks as task (task.localId)}
        <article class="upload-item">
          <div class="upload-file">
            <strong title={task.filename}>{task.filename}</strong>
            <span>
              {#if task.status === 'completed'}已完成
              {:else if task.status === 'failed'}{task.error || '上传失败'}
              {:else if task.status === 'paused'}已暂停 · {formatBytes(task.uploadedSize)} / {formatBytes(task.totalSize)}
              {:else}{task.progress}% · {formatBytes(task.speed)}/s{/if}
            </span>
          </div>
          <div class="upload-actions">
            {#if task.status === 'uploading' || task.status === 'pending'}
              <button class="icon-button" title="暂停" onclick={() => uploadQueue.pause(task.localId)}><Pause size={16} /></button>
            {:else if task.status === 'paused'}
              <button class="icon-button" title="继续" onclick={() => uploadQueue.resume(task.localId)}><Play size={16} /></button>
            {:else if task.status === 'failed'}
              <button class="icon-button" title="重试" onclick={() => uploadQueue.resume(task.localId)}><RotateCcw size={16} /></button>
            {/if}
            <button class="icon-button" title="移除任务" onclick={() => uploadQueue.remove(task.localId)}><X size={16} /></button>
          </div>
          <div class="progress-track" aria-label={`上传进度 ${task.progress}%`}>
            <span class:failed={task.status === 'failed'} style={`width:${task.progress}%`}></span>
          </div>
        </article>
      {/each}
    </div>{/if}
  </aside>
{/if}
