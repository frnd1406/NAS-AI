// NAS.AI Service Worker - Offline Support & Caching
const CACHE_NAME = 'nas-ai-v1';
const STATIC_CACHE = 'nas-ai-static-v1';
const DYNAMIC_CACHE = 'nas-ai-dynamic-v1';

// Assets to cache immediately on install
const STATIC_ASSETS = [
    '/',
    '/index.html',
    '/manifest.json',
    '/icons/icon-192x192.png',
    '/icons/icon-512x512.png',
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
    console.log('[SW] Installing Service Worker...');
    event.waitUntil(
        caches.open(STATIC_CACHE).then((cache) => {
            console.log('[SW] Caching static assets');
            return cache.addAll(STATIC_ASSETS);
        })
    );
    self.skipWaiting();
});

// Activate event - cleanup old caches
self.addEventListener('activate', (event) => {
    console.log('[SW] Activating Service Worker...');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys
                    .filter((key) => key !== STATIC_CACHE && key !== DYNAMIC_CACHE)
                    .map((key) => {
                        console.log('[SW] Removing old cache:', key);
                        return caches.delete(key);
                    })
            );
        })
    );
    self.clients.claim();
});

// Fetch event - Network first, fallback to cache
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // Skip non-GET requests
    if (request.method !== 'GET') return;

    // Skip non-HTTP schemes (chrome-extension, etc.)
    if (!url.protocol.startsWith('http')) return;

    // Skip API requests (always fetch fresh)
    if (url.pathname.startsWith('/api/')) return;

    // Skip WebSocket connections
    if (url.protocol === 'ws:' || url.protocol === 'wss:') return;

    event.respondWith(
        // Try network first
        fetch(request)
            .then((response) => {
                // Clone response for caching
                const responseClone = response.clone();

                // Cache successful responses
                if (response.status === 200) {
                    caches.open(DYNAMIC_CACHE).then((cache) => {
                        cache.put(request, responseClone);
                    });
                }

                return response;
            })
            .catch(() => {
                // Network failed, try cache
                return caches.match(request).then((cachedResponse) => {
                    if (cachedResponse) {
                        return cachedResponse;
                    }

                    // Return offline page for navigation requests
                    if (request.mode === 'navigate') {
                        return caches.match('/index.html');
                    }

                    // Return empty response for other requests
                    return new Response('Offline', {
                        status: 503,
                        statusText: 'Service Unavailable',
                    });
                });
            })
    );
});

// Handle push notifications (future feature)
self.addEventListener('push', (event) => {
    const options = {
        body: event.data?.text() || 'Neue Benachrichtigung von NAS.AI',
        icon: '/icons/icon-192x192.png',
        badge: '/icons/icon-72x72.png',
        vibrate: [100, 50, 100],
        data: {
            dateOfArrival: Date.now(),
        },
        actions: [
            { action: 'open', title: 'Öffnen' },
            { action: 'close', title: 'Schließen' },
        ],
    };

    event.waitUntil(
        self.registration.showNotification('NAS.AI', options)
    );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
    event.notification.close();

    if (event.action === 'open' || !event.action) {
        event.waitUntil(
            clients.openWindow('/')
        );
    }
});

// Background sync (for offline file uploads - future feature)
self.addEventListener('sync', (event) => {
    if (event.tag === 'sync-uploads') {
        console.log('[SW] Background sync: uploads');
        // Future: Sync queued uploads when back online
    }
});
