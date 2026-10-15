// 缓存策略或预缓存结构变化时递增版本，以便激活阶段清理旧缓存。
const CACHE_VERSION = 'v4';
const BASE_PATH = new URL(self.registration.scope).pathname; // 当前应用所在目录，保留末尾斜杠。
const CACHE_PREFIX = `octopus-${encodeURIComponent(BASE_PATH)}-`; // 按应用目录隔离缓存及其清理范围。
const CACHE_NAMES = {
    shell: `${CACHE_PREFIX}shell-${CACHE_VERSION}`,
    static: `${CACHE_PREFIX}static-${CACHE_VERSION}`,
};

// 应用壳除构建入口外还必须包含的固定 PWA 资源。
const CORE_ASSETS = [
    `${BASE_PATH}manifest.json`,
    `${BASE_PATH}favicon.ico`,
    `${BASE_PATH}apple-icon.png`,
    `${BASE_PATH}web-app-manifest-192x192.png`,
    `${BASE_PATH}web-app-manifest-512x512.png`,
];

// extractShellAssets 从构建后的 HTML 中提取根路径和相对路径资源。
function extractShellAssets(html) {
    const assets = new Set(CORE_ASSETS);
    for (const match of html.matchAll(/\b(?:href|src)=["']((?:\/|\.\/)[^"'#]+)["']/g)) {
        const url = new URL(match[1], self.registration.scope);
        assets.add(`${url.pathname}${url.search}`);
    }
    return [...assets];
}

// cacheAppShell 缓存首页、当前构建的哈希入口和固定 PWA 资源。
async function cacheAppShell() {
    const response = await fetch(BASE_PATH, { cache: 'no-store' });
    if (!response.ok) {
        throw new Error(`Failed to fetch app shell: ${response.status}`);
    }

    const assets = extractShellAssets(await response.clone().text());
    const shellCache = await caches.open(CACHE_NAMES.shell);
    const staticCache = await caches.open(CACHE_NAMES.static);
    await shellCache.put(BASE_PATH, response);
    await Promise.all([
        shellCache.addAll(assets.filter((asset) => !asset.startsWith(`${BASE_PATH}assets/`))),
        staticCache.addAll(assets.filter((asset) => asset.startsWith(`${BASE_PATH}assets/`))),
    ]);
}

self.addEventListener('install', (event) => {
    event.waitUntil((async () => {
        await cacheAppShell();
        await self.skipWaiting();
    })());
});

self.addEventListener('activate', (event) => {
    event.waitUntil((async () => {
        const currentCaches = new Set(Object.values(CACHE_NAMES));
        const cacheNames = await caches.keys();
        await Promise.all(
            cacheNames
                .filter((name) => name.startsWith(CACHE_PREFIX) && !currentCaches.has(name))
                .map((name) => caches.delete(name)),
        );
        await self.clients.claim();
    })());
});

self.addEventListener('fetch', (event) => {
    const { request } = event;
    if (request.method !== 'GET') return;

    const url = new URL(request.url);
    if (url.origin !== self.location.origin || !url.pathname.startsWith(BASE_PATH)) return;
    const pathname = url.pathname.slice(BASE_PATH.length); // 相对于应用目录的路径，用于识别 API 和静态资源。
    if (
        pathname === 'sw.js' ||
        pathname === 'api' ||
        pathname.startsWith('api/') ||
        pathname === 'v1' ||
        pathname.startsWith('v1/') ||
        pathname.startsWith('@vite') ||
        pathname.startsWith('@react-refresh')
    ) {
        return;
    }

    if (request.mode === 'navigate') {
        event.respondWith(networkFirst(request));
        return;
    }

    if (pathname.startsWith('assets/')) {
        event.respondWith(cacheFirst(request));
        return;
    }

    if (
        !['script', 'style', 'image', 'font', 'manifest'].includes(request.destination) &&
        !/\.(?:css|js|mjs|png|jpg|jpeg|webp|gif|svg|ico|woff2?|ttf|otf|json)$/i.test(url.pathname)
    ) {
        return;
    }

    const updatePromise = fetch(request).then(async (response) => {
        if (response.ok) {
            const cache = await caches.open(CACHE_NAMES.shell);
            await cache.put(request, response.clone());
        }
        return response;
    });
    event.waitUntil(updatePromise.then(() => undefined).catch(() => undefined));
    event.respondWith((async () => {
        const cache = await caches.open(CACHE_NAMES.shell);
        const cached = await cache.match(request);
        if (cached) return cached;
        try {
            return await updatePromise;
        } catch {
            return new Response('Offline', { status: 503 });
        }
    })());
});

// cacheFirst 优先返回不可变哈希资源的缓存，未命中时请求并持久化网络响应。
async function cacheFirst(request) {
    const cache = await caches.open(CACHE_NAMES.static);
    const cached = await cache.match(request);
    if (cached) return cached;

    try {
        const response = await fetch(request);
        if (response.ok) await cache.put(request, response.clone());
        return response;
    } catch {
        return new Response('Offline', { status: 503 });
    }
}

// networkFirst 优先返回最新页面，断网时回退到预缓存首页。
async function networkFirst(request) {
    const cache = await caches.open(CACHE_NAMES.shell);
    try {
        const response = await fetch(request);
        if (response.ok) await cache.put(BASE_PATH, response.clone());
        return response;
    } catch {
        return (await cache.match(request)) || (await cache.match(BASE_PATH)) || new Response('Offline', { status: 503 });
    }
}
