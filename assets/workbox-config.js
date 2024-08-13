module.exports = {
	globDirectory: '.',
	globPatterns: [
		'**/*.{html,otf,woff2,txt,woff,md,js,css,png,jpg,ijmap}'
	],
  globIgnores:[
    'index.html',
    'admin.html',
    '/'
  ],
  skipWaiting: true,
  clientsClaim: true,
	swDest: 'sw.js',
	ignoreURLParametersMatching: [
		/^utm_/,
		/^fbclid$/,
	],
  runtimeCaching: [
    {
      urlPattern: /\.(?:js|css)$/,
      handler: 'StaleWhileRevalidate'
    },
    {
      urlPattern: /\//,
      handler: 'NetworkFirst',
    },
    {
      urlPattern: /\/admin/,
      handler: 'NetworkFirst',
    }
  ],
};
