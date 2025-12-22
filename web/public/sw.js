// Service Worker for Octopus PWA
const CACHE_VERSION = 'v1';
const CACHE_NAME = `octopus-cache-${CACHE_VERSION}`;
const RUNTIME_CACHE = `octopus-runtime-${CACHE_VERSION}`;

// 需要预缓存的静态资源
const PRECACHE_URLS = [
    '/',
    '/manifest.json',
    '/web-app-manifest-192x192.png',
    '/web-app-manifest-512x512.png',
    '/logo-dark.svg',
];

// 安装事件 - 预缓存资源
self.addEventListener('install', (event) => {
    console.log('[SW] Install event');
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            console.log('[SW] Precaching static resources');
            return cache.addAll(PRECACHE_URLS).catch((err) => {
                console.error('[SW] Precache failed:', err);
            });
        }).then(() => {
            // 强制跳过等待，立即激活
            return self.skipWaiting();
        })
    );
});

// 激活事件 - 清理旧缓存
self.addEventListener('activate', (event) => {
    console.log('[SW] Activate event');
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames.map((cacheName) => {
                    // 删除旧版本的缓存
                    if (cacheName !== CACHE_NAME && cacheName !== RUNTIME_CACHE) {
                        console.log('[SW] Deleting old cache:', cacheName);
                        return caches.delete(cacheName);
                    }
                })
            );
        }).then(() => {
            // 立即接管所有页面
            return self.clients.claim();
        })
    );
});

// Fetch 事件 - 网络请求拦截
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // 只处理同源请求
    if (url.origin !== location.origin) {
        return;
    }

    // 跳过某些请求
    if (
        request.method !== 'GET' ||
        url.pathname.startsWith('/api/') ||
        url.pathname.startsWith('/_next/webpack-hmr')
    ) {
        return;
    }

    event.respondWith(
        caches.match(request).then((cachedResponse) => {
            // 如果缓存中有，先返回缓存
            if (cachedResponse) {
                // 同时在后台更新缓存 (Stale-While-Revalidate 策略)
                fetch(request).then((networkResponse) => {
                    if (networkResponse && networkResponse.status === 200) {
                        caches.open(RUNTIME_CACHE).then((cache) => {
                            cache.put(request, networkResponse);
                        });
                    }
                }).catch(() => {
                    // 网络请求失败时忽略
                });

                return cachedResponse;
            }

            // 缓存中没有，从网络获取
            return fetch(request).then((networkResponse) => {
                // 检查是否是有效响应
                if (!networkResponse || networkResponse.status !== 200 || networkResponse.type === 'error') {
                    return networkResponse;
                }

                // 克隆响应，因为响应流只能使用一次
                const responseToCache = networkResponse.clone();

                // 缓存资源（仅缓存页面和静态资源）
                const isFont = request.destination === 'font' || /\.(woff2?|ttf|otf|eot)$/i.test(url.pathname);
                if (
                    request.destination === 'document' ||
                    request.destination === 'script' ||
                    request.destination === 'style' ||
                    request.destination === 'image' ||
                    request.destination === 'font'
                ) {
                    caches.open(RUNTIME_CACHE).then((cache) => {
                        cache.put(request, responseToCache);
                    });
                }

                return networkResponse;
            }).catch((error) => {
                console.error('[SW] Fetch failed:', error);

                // 如果是导航请求（页面），返回离线页面
                if (request.destination === 'document') {
                    return caches.match('/');
                }

                throw error;
            });
        })
    );
});

// 消息事件 - 支持手动触发缓存更新
self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }

    if (event.data && event.data.type === 'CACHE_URLS') {
        event.waitUntil(
            caches.open(RUNTIME_CACHE).then((cache) => {
                return cache.addAll(event.data.payload);
            })
        );
    }
});
