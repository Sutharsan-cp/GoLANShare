self.addEventListener('install', (e) => {
  e.waitUntil(
    caches.open('golanshare-v1').then((cache) => cache.addAll([
      '/',
      '/index.html',
      '/js/app.js',
      '/css/style.css',
    ])),
  );
});

self.addEventListener('fetch', (e) => {
  e.respondWith(
    caches.match(e.request).then((response) => response || fetch(e.request)),
  );
});