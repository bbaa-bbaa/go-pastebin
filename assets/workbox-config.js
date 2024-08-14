module.exports = {
	globDirectory: '.',
	globPatterns: [
		'**/*.{html,otf,woff2,txt,woff,md,js,css,png,jpg,ico,ijmap}',
                'manifest.json'
	],
  globIgnores:[
    'index.html',
    'admin.html'
  ],
  skipWaiting: true,
  clientsClaim: true,
	swDest: 'sw.js',
	ignoreURLParametersMatching: [
		/^utm_/,
		/^fbclid$/,
	],
  mode: "production",
  runtimeCaching: [
    {
      urlPattern: function(options){
        const url = options.url;
        return url.pathname.startsWith('/static/') && (url.pathname.endsWith('.js') || url.pathname.endsWith('.css'));
      },
      handler: 'StaleWhileRevalidate'
    },
    {
      urlPattern: function(options){
        const url = options.url;
        return url.pathname === '/' || url.pathname === '/admin/';
      },
      handler: 'NetworkFirst',
    }
  ],
};
