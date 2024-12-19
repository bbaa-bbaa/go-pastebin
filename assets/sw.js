const CACHE_NAME = "pastebin-cache-v1";
const RUNTIME_CACHE_NAME = "pastebin-runtime-cache-v1";
const networkTimeout = 5000;
let runtimeCache = caches.open(RUNTIME_CACHE_NAME);
let persistCache = caches.open(CACHE_NAME);

async function requestCached(request, cache) {
  return cache.match(request).then(function (cached) {
    if (cached) {
      return true;
    }
    return false;
  });
}

async function cleanOldCacheStorage() {
  return caches.keys().then(async function (cacheNames) {
    for (let cacheName of cacheNames) {
      if (cacheName !== CACHE_NAME && cacheName !== RUNTIME_CACHE_NAME) {
        await caches.delete(cacheName);
      }
    }
  });
}

async function cleanRuntimeCacheInPersist() {
  return runtimeCache.then(async function (cache) {
    return cache.keys().then(async function (requests) {
      for (let request of requests) {
        if (requestCached(request, await persistCache)) {
          cache.delete(request);
        }
      }
    });
  });
}

async function networkFirst(cache, response) {
  if (!cache) return await response;
  return Promise.race([
    response,
    new Promise(function (resolve, reject) {
      setTimeout(function () {
        resolve(cache);
      }, networkTimeout);
    })
  ]);
}

// return updated
async function updatePersistCache() {
  let updated = false;
  return fetch("api/sw/manifest/v1")
    .then(async function (response) {
      return response.json();
    })
    .then(async function (manifest) {
      let cache = await persistCache;
      let pendingRequests = [];
      for (let path of Object.keys(manifest.hash)) {
        let cached_response = await cache.match(path);
        if (!cached_response) {
          updated = true;
          pendingRequests.push(
            fetch(path).then(async function (response) {
              cache.put(path, response);
            })
          );
        } else {
          let etag = cached_response.headers.get("ETag");
          if (manifest.hash[path] !== etag) {
            updated = true;
            pendingRequests.push(
              fetch(path).then(async function (response) {
                cache.put(path, response);
              })
            );
          }
        }
      }
      return Promise.all(pendingRequests);
    })
    .then(() => updated)
    .catch(() => false);
}

async function cacheIndex() {
  return fetch("/").then(async function (response) {
    let cache = await runtimeCache;
    return cache.put("/", response);
  });
}

self.addEventListener("install", function (event) {
  self.skipWaiting();
  event.waitUntil(cleanOldCacheStorage().then(updatePersistCache).then(cleanRuntimeCacheInPersist).then(cacheIndex));
});

self.addEventListener("fetch", function (event) {
  let url = URL.parse(event.request.url);
  if (url.pathname == "/") {
    updatePersistCache()
      .then(function (updated) {
        if (updated) {
          return clients.matchAll();
        }
        return [];
      })
      .then(function (clients) {
        for (let client of clients) {
          client.postMessage({ type: "update" });
        }
      });
  }
  event.respondWith(
    caches.match(event.request).then(function (cached) {
      let response = fetch(event.request)
        .then(async function (response) {
          if (event.request.method == "GET") {
            if (url.pathname == "/" || (url.pathname.split("/").length > 2 && parseInt(response.headers.get("Content-Length")) < 5 * 1048576)) {
              if (!(await requestCached(event.request, await persistCache))) {
                let clone_respone = response.clone();
                runtimeCache.then(function (cache) {
                  cache.put(event.request, clone_respone);
                });
              }
            }
          }
          return response;
        })
        .catch(function () {
          if (!cached) {
            return new Response(
              `
            <!DOCTYPE html>
            <html lang="en">
              <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Network Error</title>
              </head>
              <body>
                <h1>Network Error</h1>
                <p>Unable to connect to the server. Please check your network connection.</p>
                <p>message from service worker.</p>
              </body>
            </html>
          `,
              {
                status: 503,
                statusText: "Service Unavailable",
                headers: {
                  "Content-Type": "text/html"
                }
              }
            );
          }
          return cached;
        });
      return networkFirst(cached, response);
    })
  );
});

self.addEventListener("activate", function (event) {
  let resource_version_updated = false;
  event.waitUntil(
    clients
      .claim()
      .then(cleanOldCacheStorage())
      .then(updatePersistCache)
      .then(updated => {
        resource_version_updated = updated;
      })
      .then(cleanRuntimeCacheInPersist)
      .then(() => {
        if (resource_version_updated) {
          return clients.matchAll();
        }
        return [];
      })
      .then(function (clients) {
        for (let client of clients) {
          client.postMessage({ type: "update" });
        }
      })
  );
});
