const CACHE_NAME = "pastebin-cache-v1";
const RUNTIME_CACHE_NAME = "pastebin-runtime-cache-v1";
const networkTimeout = 5000;
const runtimeCacheLifetime = 7 * 86400;
const runtimeCacheMaxSize = 1048576;
let messageBus = [];

async function runtimeCache() {
  return caches.open(RUNTIME_CACHE_NAME);
}
async function persistCache() {
  return caches.open(CACHE_NAME);
}

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

async function cleanUnusePersistCache(manifest) {
  return persistCache().then(async function (cache) {
    let cachedPath = Object.keys(manifest.hash);
    return cache.keys().then(async function (requests) {
      for (let request of requests) {
        let url = URL.parse(request.url);
        let path = url.pathname;
        if (path.length >= 1 && path[0] === "/") {
          path = path.slice(1);
        }
        if (!cachedPath.includes(path)) {
          console.log("cleanUnusePersistCache", request.url);
          cache.delete(request);
        }
      }
    });
  });
}

async function cleanRuntimeCacheInPersist() {
  return runtimeCache().then(async function (cache) {
    return cache.keys().then(async function (requests) {
      for (let request of requests) {
        if (await requestCached(request, await persistCache())) {
          console.log("cleanRuntimeCacheInPersist", request.url);
          cache.delete(request);
        }
      }
    });
  });
}

async function cleanOutdatedRuntimeCache() {
  return runtimeCache().then(async function (cache) {
    return cache.keys().then(async function (requests) {
      for (let request of requests) {
        let url = URL.parse(request.url);
        if (url.pathname == "/") continue;
        let response = await cache.match(request);
        let lastAccess = new Date(response.headers.get("X-Last-Access") || 0);
        if (Date.now() - lastAccess.getTime() > runtimeCacheLifetime * 1000) {
          cache.delete(request);
        }
      }
    });
  });
}

async function getResponseSize(response) {
  let size = 0;
  if (response.headers.has("Content-Length")) {
    size = parseInt(response.headers.get("Content-Length"));
  } else {
    let body = response.body.getReader();
    while (true) {
      let { done, value } = await body.read();
      if (done) break;
      size += value.length;
    }
  }
  return size;
}

async function networkFirst(cache, response) {
  if (!cache) return response;
  return Promise.race([
    response,
    new Promise(function (resolve) {
      setTimeout(function () {
        resolve(cache);
      }, networkTimeout);
    })
  ]);
}

// return updated
async function updatePersistCache() {
  let updated = false;
  return fetch(new Request("api/sw/v1/manifest", { cache: "no-store" }))
    .then(async function (response) {
      return response.json();
    })
    .then(async function (manifest) {
      let cache = await persistCache();
      let pendingRequests = [];
      for (let path of Object.keys(manifest.hash)) {
        let cached_response = await cache.match(path);
        if (!cached_response) {
          console.log("detected new resource", path);
          updated = true;
          pendingRequests.push(
            fetch(new Request(path, { cache: "no-store" })).then(async function (response) {
              return cache.put(path, response);
            })
          );
        } else {
          let revision = cached_response.headers.get("X-Revision") || cached_response.headers.get("ETag");
          if (manifest.hash[path] !== revision) {
            console.log("detected updated resource", path);
            updated = true;
            pendingRequests.push(
              fetch(new Request(path, { cache: "no-store" })).then(async function (response) {
                return cache.put(path, response);
              })
            );
          }
        }
      }
      return Promise.all(pendingRequests).then(() => {
        cleanUnusePersistCache(manifest);
      });
    })
    .then(() => updated)
    .catch(e => {
      console.log(e);
      return false;
    });
}

async function addLastAccess(response) {
  let headers = new Headers(response.headers);
  headers.set("X-Last-Access", new Date().toUTCString());
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: headers
  });
}

async function addToRuntimeCache(request, response) {
  if (!(await requestCached(request, await persistCache()))) {
    runtimeCache()
      .then(async function (cache) {
        let url = URL.parse(request.url);
        if (url.protocol !== "http:" && url.protocol !== "https:") return;
        return cache.put(request, await addLastAccess(response));
      })
      .catch(() => {});
  }
}

async function cacheIndex() {
  return fetch("/").then(async function (response) {
    addToRuntimeCache(new Request("/"), response);
  });
}

// reliable message channel
// all clients should ack the message
// if a client is not acked, the message will be sent again
let reliableMessageId = 0;
async function reliableMessage(data) {
  return new Promise(async function (resolve) {
    let msgid = reliableMessageId++;
    let ackedSeq = [];
    let clientList = (await clients.matchAll()).map(c => c.id);
    let messageAckedChannel = function (event) {
      if (event.data.megseq !== "") {
        if (event.data.msgid == msgid) {
          ackedSeq.push(event.data.msgseq);
          console.log(`msgseq: "${event.data.msgseq}" acked`);
        }
      }
    };

    messageBus.push(messageAckedChannel);

    let intervalId = setInterval(() => {
      postMessage();
    }, 1000);

    function complete() {
      clearInterval(intervalId);
      let index = messageBus.indexOf(messageAckedChannel);
      if (index >= 0) {
        messageBus.splice(index, 1);
      }
      resolve();
    }

    async function postMessage() {
      let ackedCount = 0;
      for (let clientId of clientList) {
        let client = await clients.get(clientId);
        if (!client) {
          ackedCount++;
          continue;
        }
        if (!ackedSeq.includes(`${msgid}/${client.id}`)) {
          client.postMessage(Object.assign(data, { msgseq: `${msgid}/${client.id}`, msgid: msgid }));
          console.log(`send msg type: "${data.type}" seq:"${msgid}/${client.id}"`);
        } else {
          ackedCount++;
        }
      }
      if (ackedCount >= clientList.length) {
        complete();
      }
    }
  });
}

async function notifyUpdate() {
  return reliableMessage({ type: "update" });
}

self.addEventListener("message", function (event) {
  for (let bus of messageBus) {
    bus(event);
  }
});

messageBus.push(function (event) {
  if (event.data.type == "check-update") {
    console.log("check update");
    updatePersistCache().then(async function (updated) {
      await cleanRuntimeCacheInPersist();
      if (updated) {
        return notifyUpdate();
      } else {
        if (event.source) event.source.postMessage({ type: "up-to-date" });
      }
    });
  }
});

self.addEventListener("install", function (event) {
  self.skipWaiting();
  event.waitUntil(cleanOldCacheStorage());
});

self.addEventListener("fetch", function (event) {
  if (event.request.method !== "GET") {
    return; // bypass non-GET request
  }
  event.respondWith(
    persistCache().then(persist_cache => {
      return persist_cache.match(event.request).then(response => {
        if (response) {
          return response;
        }
        return runtimeCache()
          .then(runtime_cache => {
            return runtime_cache.match(event.request);
          })
          .then(function (cached) {
            let response = fetch(event.request)
              .then(function (response) {
                getResponseSize(response.clone()).then(
                  (response =>
                    function (size) {
                      if (size <= runtimeCacheMaxSize) {
                        addToRuntimeCache(event.request, response);
                      }
                    })(response.clone())
                );
                return response;
              })
              .catch(function () {
                if (!cached) {
                  return new Response(
                    `<!DOCTYPE html>
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
</html>`,
                    {
                      status: 503,
                      statusText: "Service Unavailable",
                      headers: {
                        "Content-Type": "text/html"
                      }
                    }
                  );
                }
                addToRuntimeCache(event.request, cached.clone());
                return cached;
              });
            return networkFirst(cached, response);
          });
      });
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
      .then(cleanOutdatedRuntimeCache)
      .then(cacheIndex)
      .then(() => {
        if (resource_version_updated) {
          return notifyUpdate();
        }
      })
  );
});
