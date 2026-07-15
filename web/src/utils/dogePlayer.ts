const DOGE_PLAYER_SCRIPT = 'https://player.dogecloud.com/js/loader'
let dogePlayerLoader: Promise<void> | null = null

export type DogePlayerOptions = {
    container: HTMLDivElement
    vcode: string
    userId: number
    autoPlay?: boolean
}

export type DogePlayerInstance = {
    destroy?: () => void
}

type DogePlayerConstructor = new (options: DogePlayerOptions) => DogePlayerInstance

type DogePlayerWindow = Window & {
    DogePlayer?: DogePlayerConstructor
    DogeCloudPlayer?: DogePlayerConstructor
    default?: DogePlayerConstructor
}

export function resolveDogePlayer(): DogePlayerConstructor | null {
    const playerWindow = window as DogePlayerWindow
    return playerWindow.DogePlayer || playerWindow.DogeCloudPlayer || playerWindow.default || null
}

function waitForDogePlayer(timeoutMs = 5000) {
    const start = Date.now()
    return new Promise<void>((resolve, reject) => {
        const tick = () => {
            if (resolveDogePlayer()) {
                resolve()
                return
            }
            if (Date.now() - start >= timeoutMs) {
                reject(new Error('播放器未就绪'))
                return
            }
            window.setTimeout(tick, 100)
        }
        tick()
    })
}

export function loadDogePlayerScript() {
    if (dogePlayerLoader) return dogePlayerLoader

    const loaderPromise = new Promise<void>((resolve, reject) => {
        if (resolveDogePlayer()) {
            resolve()
            return
        }

        const existing = document.querySelector('script[data-doge-player-sdk="true"]') as HTMLScriptElement | null
        if (existing) {
            existing.remove()
        }

        const script = document.createElement('script')
        script.type = 'text/javascript'
        script.src = DOGE_PLAYER_SCRIPT
        script.setAttribute('data-doge-player-sdk', 'true')
        script.onload = () => {
            waitForDogePlayer().then(resolve).catch(reject)
        }
        script.onerror = () => reject(new Error('播放器脚本加载失败'))
        document.head.appendChild(script)
    }).catch((error) => {
        dogePlayerLoader = null
        throw error
    })

    dogePlayerLoader = loaderPromise
    return dogePlayerLoader
}

export function firstNonEmptyString(...values: Array<string | undefined | null>) {
    for (const value of values) {
        const v = value?.trim()
        if (v) return v
    }
    return ''
}

export function playerUserIDFromPlayURL(raw?: string) {
    const value = raw?.trim()
    if (!value) return ''
    try {
        const u = new URL(value)
        return firstNonEmptyString(
            u.searchParams.get('userId') ?? '',
            u.searchParams.get('userid') ?? '',
            u.searchParams.get('uid') ?? '',
        )
    } catch {
        return ''
    }
}

export function buildDogeShareURL(
    vcode: string,
    userId: string,
    options?: { autoPlay?: boolean; inFrame?: boolean },
) {
    const nextVCode = vcode.trim()
    const nextUserId = userId.trim()
    if (!nextVCode || !nextUserId) return ''

    const params = new URLSearchParams({
        vcode: nextVCode,
        userId: nextUserId,
    })
    if (options?.autoPlay) params.set('autoPlay', 'true')
    if (options?.inFrame) params.set('inFrame', 'true')
    return `https://player.dogecloud.com/web/player.html?${params.toString()}`
}

export function buildDogeIframeCode(input: {
    vcode: string
    userId: string
    autoPlay: boolean
    width: string
    height: string
}) {
    const src = buildDogeShareURL(input.vcode, input.userId, {
        autoPlay: input.autoPlay,
        inFrame: true,
    })
    return `<iframe id="dogePlayerFrame" src="${src}" allowfullscreen="true" msallowfullscreen="true" webkitallowfullscreen="true" mozallowfullscreen="true" oallowfullscreen="true" allowtransparency="true" scrolling="no" width="${input.width}" height="${input.height}" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture; fullscreen" referrerPolicy="unsafe-url"></iframe>`
}

export function normalizeDimension(value: string, fallback: string) {
    const next = value.trim()
    return next || fallback
}
