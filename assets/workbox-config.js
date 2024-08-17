const fs = require("fs");
const crypto = require("crypto");

const integrityManifestTransform = originalManifest => {
  const manifest = originalManifest.map(entry => {
    entry.integrity = `sha256-${crypto.createHash("sha256").update(fs.readFileSync(entry.url)).digest("base64")}`;
    return entry;
  });
  return { manifest };
};

module.exports = {
  globDirectory: ".",
  globPatterns: [
    "**/*.{js,css,ico}",
    "static/mdui/fonts/roboto/{Roboto-Regular.woff2,Roboto-Medium.woff2,Roboto-Bold.woff2}",
    "static/mdui/icons/material-icons/MaterialIcons-Regular.woff2",
    "static/font/Bender/Bender-Bold.woff2",
    "static/font/Hack/hack-regular.woff2"
  ],
  /*
  globIgnores:[
    'index.html',
    'admin.html'
  ],
  */
  skipWaiting: true,
  clientsClaim: true,
  cleanupOutdatedCaches: true,
  manifestTransforms: [integrityManifestTransform],
  swDest: "sw.js",
  ignoreURLParametersMatching: [/^utm_/, /^fbclid$/],
  mode: "production",
  runtimeCaching: [
    {
      urlPattern: function (options) {
        const url = options.url;
        return (
          url.pathname.startsWith("/static/img/") ||
          (url.pathname.startsWith("/static/") &&
            (url.pathname.endsWith(".js") || url.pathname.endsWith(".css") || url.pathname.endsWith(".woff") || url.pathname.endsWith(".woff2")))
        );
      },
      handler: "StaleWhileRevalidate"
    },
    {
      urlPattern: function (options) {
        const url = options.url;
        return url.pathname === "/" || url.pathname == "manifest.json" || url.pathname === "/api/user" || url.pathname === "/admin/";
      },
      handler: "NetworkFirst"
    }
  ]
};
